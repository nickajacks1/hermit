package hermittest

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/alecthomas/assert/v2"
	"github.com/cashapp/hermit"
	"github.com/cashapp/hermit/cache"
	"github.com/cashapp/hermit/envars"
	"github.com/cashapp/hermit/internal/dao"
	"github.com/cashapp/hermit/sources"
	"github.com/cashapp/hermit/state"
	"github.com/cashapp/hermit/ui"
	"github.com/cashapp/hermit/vfs"
)

func makeWritable(path string) error {
	return filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		return os.Chmod(path, info.Mode()|0200) // Add write permission
	})
}

// EnvTestFixture encapsulates the directories used by Env and the Env itself
type EnvTestFixture struct {
	State   *state.State
	EnvDirs []string
	Env     *hermit.Env
	Logs    *bytes.Buffer
	Server  *httptest.Server
	P       *ui.UI
	t       *testing.T
	Cache   *cache.Cache
}

// NewEnvTestFixture returns a new empty fixture with Env initialised to default values.
// A test handler can be given to be used as an test http server for testing http interactions
func NewEnvTestFixture(t *testing.T, handler http.Handler) *EnvTestFixture {
	t.Helper()
	envDir := t.TempDir()
	stateDir := t.TempDir()

	log, buf := ui.NewForTesting()

	err := hermit.Init(log, envDir, "", stateDir, hermit.Config{}, "BYPASS")
	assert.NoError(t, err)

	server := httptest.NewServer(handler)
	client := server.Client()
	cache, err := cache.Open(stateDir, nil, client, client)
	assert.NoError(t, err)
	sta, err := state.Open(stateDir, state.Config{
		Sources: []string{},
		Builtin: sources.NewBuiltInSource(vfs.InMemoryFS(nil)),
	}, cache)
	assert.NoError(t, err)
	info, err := hermit.LoadEnvInfo(envDir)
	assert.NoError(t, err)
	env, err := hermit.OpenEnv(info, sta, cache.GetSource, envars.Envars{}, server.Client(), nil)
	assert.NoError(t, err)

	fixture := &EnvTestFixture{
		Cache:   cache,
		State:   sta,
		EnvDirs: []string{envDir},
		Logs:    buf,
		Env:     env,
		Server:  server,
		t:       t,
		P:       log,
	}

	// Register cleanup function that makes files writable before removal
	t.Cleanup(func() {
		_ = makeWritable(fixture.RootDir())
		fixture.Clean()
	})

	return fixture
}

func (f *EnvTestFixture) ScriptSums() []string {
	var sums []string
	for _, file := range []string{"activate-hermit", "activate-hermit.fish", "hermit"} {
		path := filepath.Join(f.Env.BinDir(), file)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			continue
		}
		hasher := sha256.New()
		r, err := os.Open(path)
		if err != nil {
			continue
		}
		_, _ = io.Copy(hasher, r)
		_ = r.Close()
		sums = append(sums, hex.EncodeToString(hasher.Sum(nil)))
	}
	return sums
}

// RootDir returns the directory to the environment package root
func (f *EnvTestFixture) RootDir() string {
	return filepath.Join(f.State.Root(), "pkg")
}

// DAO returns the DAO using the underlying hermit database
func (f *EnvTestFixture) DAO() *dao.DAO {
	d, err := dao.Open(f.State.Root())
	assert.NoError(f.t, err)
	return d
}

// Clean removes all files and directories from this environment
func (f *EnvTestFixture) Clean() {
	for _, dir := range f.EnvDirs {
		os.RemoveAll(dir)
	}
	os.RemoveAll(f.State.Root())
	f.Server.Close()
}

// NewEnv returns a new environment using the state directory from this fixture
func (f *EnvTestFixture) NewEnv() *hermit.Env {
	envDir := f.t.TempDir()
	log, _ := ui.NewForTesting()
	err := hermit.Init(log, envDir, "", f.State.Root(), hermit.Config{}, "BYPASS")
	assert.NoError(f.t, err)
	info, err := hermit.LoadEnvInfo(envDir)
	assert.NoError(f.t, err)
	env, err := hermit.OpenEnv(info, f.State, f.Cache.GetSource, envars.Envars{}, f.Server.Client(), nil)
	assert.NoError(f.t, err)
	return env
}

// GetDBPackage return the data from the DB for a package
func (f *EnvTestFixture) GetDBPackage(ref string) *dao.Package {
	dao := f.DAO()
	dbPkg, err := dao.GetPackage(ref)
	assert.NoError(f.t, err)
	return dbPkg
}

// WithManifests sets the resolver manifests for the current environment.
// Warning: any additional environments created from this fixture previously
// will not be updated.
func (f *EnvTestFixture) WithManifests(files map[string]string) *EnvTestFixture {
	for name, content := range files {
		err := f.Env.AddSource(f.P, sources.NewMemSource(name, content))
		assert.NoError(f.t, err)
	}
	return f
}
