package gocense

import (
	"archive/zip"
	"bufio"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/KyleBanks/depth"
	"github.com/go-resty/resty/v2"
	"github.com/pkg/errors"
)

// Service is a gocense service
type Service interface {
	Get(path string) []depth.Pkg
	GetAllLicenses(deps []depth.Pkg) (map[string][]byte, error)
}

type service struct {
	restyClient *resty.Client
}

func New() Service {
	return &service{
		restyClient: resty.New(),
	}
}

var ErrDownloadLicense = errors.New("could not download license")

func (s *service) GetAllLicenses(deps []depth.Pkg) (map[string][]byte, error) {
	const errMessage = "could not fetch licenses"

	result := make(map[string][]byte)

	// https://raw.githubusercontent.com/go-resty/resty/master/LICENSE
	// github.com/interesanter/pfad
	// github.com/foo1/foo2/foo3

	for i := range deps {
		dep := deps[i]

		if !strings.Contains(dep.Name, "github") {
			println("Hi i'm mr not GitHub, LOOK AT MEEEE!!!!", dep.Name)
			licenseData, err := getFromOtherSources(dep.Name)
			if err != nil {
				return nil, errors.Wrap(err, errMessage)
			}

			result[dep.Name] = licenseData
			continue
		}

		splittedPath := strings.Split(dep.Name, "/")
		if len(splittedPath) > 3 {
			continue
		}

		path := "https://" + dep.Name
		path += "/master/LICENSE"

		path = strings.Replace(path, "github.com", "raw.githubusercontent.com", 1)

		resp, err := s.restyClient.R().Get(path)
		if err != nil {
			return nil, errors.Wrap(err, errMessage)
		}

		if resp.IsError() {
			return nil, errors.Wrap(ErrDownloadLicense, errMessage)
		}

		result[dep.Name] = resp.Body()
	}

	return result, nil
}

func getFromOtherSources(path string) ([]byte, error) {
	const errMessage = "could not get license file from other sources"

	cmd := exec.Command("go", "get", "-u", "-v", path)
	err := cmd.Run()
	if err != nil {
		return nil, errors.Wrap(err, errMessage)
	}

	gopath := os.Getenv("GOPATH")
	gopath = strings.Replace(gopath, ";", "", 1)

	listPath := filepath.Join(gopath, "pkg", "mod", "cache", "download", path, "@v", "list")

	file, err := os.Open(listPath)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	latestVersion := ""
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		latestVersion = scanner.Text()
	}

	fileName := latestVersion + ".zip"
	zipPath := filepath.Join(gopath, "pkg", "mod", "cache", "download", path, "@v", fileName)

	licenseData, err := unzip(zipPath, "LICENSE")
	if err != nil {
		return nil, errors.Wrap(err, errMessage)
	}

	return licenseData, nil
}

func unzip(source string, fileName string) ([]byte, error) {
	const errMessage = "could not unzip file"

	reader, err := zip.OpenReader(source)
	if err != nil {
		return nil, errors.Wrap(err, errMessage)
	}
	defer reader.Close()

	for _, file := range reader.File {
		if file.Mode().IsDir() {
			continue
		}

		if !strings.Contains(file.Name, "LICENSE") {
			continue
		}

		licenseFile, err := file.Open()
		if err != nil {
			return nil, errors.Wrap(err, errMessage)
		}

		licenseData, err := ioutil.ReadAll(licenseFile)
		if err != nil {
			return nil, errors.Wrap(err, errMessage)
		}

		err = licenseFile.Close()
		if err != nil {
			return nil, errors.Wrap(err, errMessage)
		}

		return licenseData, nil
	}

	return nil, nil
}

func (s *service) Get(path string) []depth.Pkg {
	var t depth.Tree
	t.MaxDepth = 1000
	t.ResolveTest = false
	t.ResolveInternal = true

	err := t.Resolve(path)
	if err != nil {
		log.Fatal(err)
	}

	var result []depth.Pkg
	result = findAllDependencies(t.Root.Deps, result)

	return result
}

func findAllDependencies(deps []depth.Pkg, result []depth.Pkg) []depth.Pkg {
	for i := range deps {
		dep := deps[i]

		if dep.Internal {
			continue
		}

		if strings.Contains(dep.Name, "golang.org/x") {
			continue
		}

		if !containsDep(dep, result) {
			result = append(result, dep)
		}

		if len(dep.Deps) == 0 {
			continue
		}

		result = appendIfNotExist(result, findAllDependencies(dep.Deps, result))
	}

	return result
}

func appendIfNotExist(currentDeps, newDeps []depth.Pkg) []depth.Pkg {
	for i := range newDeps {
		newDep := newDeps[i]

		contains := containsDep(newDep, currentDeps)
		if !contains {
			currentDeps = append(currentDeps, newDep)
		}
	}

	return newDeps
}

func containsDep(dep depth.Pkg, deps []depth.Pkg) bool {
	for j := range deps {
		currentDep := deps[j]

		if currentDep.Name == dep.Name {
			return true
		}
	}

	return false
}
