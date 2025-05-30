package app

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/alecthomas/colour"

	"github.com/cashapp/hermit"
	"github.com/cashapp/hermit/envars"
	"github.com/cashapp/hermit/errors"
	"github.com/cashapp/hermit/manifest"
	"github.com/cashapp/hermit/shell"
	"github.com/cashapp/hermit/state"
	"github.com/cashapp/hermit/ui"
)

type infoCmd struct {
	Packages []manifest.GlobSelector `arg:"" required:"" help:"Packages to retrieve information for" predictor:"package"`
	JSONFormattable
}

func (i *infoCmd) Run(l *ui.UI, env *hermit.Env, sta *state.State) error {
	var installed map[string]*manifest.Package
	var err error
	packages := []*manifest.Package{}
	for _, selector := range i.Packages {
		var pkg *manifest.Package
		if env != nil {
			if installed == nil {
				installed, err = getInstalledPackageMap(l, env)
				if err != nil {
					return errors.WithStack(err)
				}
			}
			// If the selector is an exact package name match with an installed package we'll just use it.
			if pkg = installed[selector.String()]; pkg == nil {
				pkg, err = env.Resolve(l, selector, false)
				if err != nil {
					return errors.WithStack(err)
				}
			}

		} else {
			pkg, err = sta.Resolve(l, selector)
			if err != nil {
				return errors.WithStack(err)
			}
		}
		packages = append(packages, pkg)
	}

	envroot := "<env>" // Used as a place holder in env vars if there is no active environment
	if env != nil {
		envroot = env.Root()
	}

	if i.JSON {
		js, err := json.Marshal(packages) //nolint:musttag // default JSON behavior is fine
		if err != nil {
			return errors.WithStack(err)
		}
		l.Printf("%s\n", string(js))
		return nil
	}

	for j, pkg := range packages {
		colour.Printf("^B^2Name:^R %s\n", pkg.Reference.Name)
		if pkg.Reference.Version.IsSet() {
			colour.Printf("^B^2Version:^R %s\n", pkg.Reference.Version)
		} else {
			colour.Printf("^B^2Channel:^R %s\n", pkg.Reference.Channel)
		}
		colour.Printf("^B^2Description:^R %s\n", pkg.Description)
		colour.Printf("^B^2Homepage:^R %s\n", pkg.Homepage)
		colour.Printf("^B^2State:^R %s\n", pkg.State)
		colour.Printf("^B^2Source:^R %s\n", pkg.Source)
		colour.Printf("^B^2Root:^R %s\n", pkg.Root)
		if len(pkg.Requires) != 0 {
			colour.Printf("^B^2Requires:^R %s\n", strings.Join(pkg.Requires, " "))
		}
		if len(pkg.Provides) != 0 {
			colour.Printf("^B^2Provides:^R %s\n", strings.Join(pkg.Provides, " "))
		}
		environ := envars.Parse(os.Environ()).Apply(envroot, pkg.Env).Changed(false)
		if len(environ) != 0 {
			colour.Printf("^B^2Envars:^R\n")
			for key, value := range environ {
				colour.Printf("  %s=%s\n", key, shell.Quote(value))
			}
		}
		bins, _ := pkg.ResolveBinaries()
		for i := range bins {
			bins[i] = filepath.Base(bins[i])
		}
		if len(bins) > 0 {
			colour.Printf("^B^2Binaries:^R %s\n", strings.Join(bins, " "))
		}
		if j < len(i.Packages)-1 {
			colour.Printf("\n")
		}
	}
	return nil
}

func getInstalledPackageMap(l *ui.UI, env *hermit.Env) (map[string]*manifest.Package, error) {
	installedPkgs, err := env.ListInstalled(l)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	installed := make(map[string]*manifest.Package, len(installedPkgs))
	for _, installedPkg := range installedPkgs {
		installed[installedPkg.Reference.Name] = installedPkg
	}
	return installed, nil
}
