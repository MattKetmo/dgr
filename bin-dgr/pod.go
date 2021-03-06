package main

import (
	"github.com/blablacar/dgr/bin-dgr/common"
	"github.com/ghodss/yaml"
	"github.com/n0rad/go-erlog/data"
	"github.com/n0rad/go-erlog/errs"
	"github.com/n0rad/go-erlog/logs"
	"io/ioutil"
	"path/filepath"
)

const pathPodManifestYml = "/pod-manifest.yml"

type Pod struct {
	fields   data.Fields
	path     string
	args     BuildArgs
	target   string
	manifest common.PodManifest
}

func NewPod(path string, args BuildArgs) (*Pod, error) {
	fullPath, err := filepath.Abs(path)
	if err != nil {
		logs.WithE(err).WithField("path", path).Fatal("Cannot get fullpath")
	}

	manifest, err := readPodManifest(fullPath + pathPodManifestYml)
	if err != nil {
		manifest2, err2 := readPodManifest(fullPath + "/cnt-pod-manifest.yml")
		if err2 != nil {
			return nil, errs.WithEF(err, data.WithField("path", fullPath+pathPodManifestYml).WithField("err2", err2), "Failed to read pod manifest")
		}
		logs.WithField("old", "cnt-pod-manifest.yml").WithField("new", "pod-manifest.yml").Warn("You are using the old aci configuration file")
		manifest = manifest2
	}
	fields := data.WithField("pod", manifest.Name.String())

	target := path + pathTarget
	if Home.Config.TargetWorkDir != "" {
		currentAbsDir, err := filepath.Abs(Home.Config.TargetWorkDir + "/" + manifest.Name.ShortName())
		if err != nil {
			logs.WithEF(err, fields).Panic("invalid target path")
		}
		target = currentAbsDir
	}

	pod := &Pod{
		fields:   fields,
		path:     fullPath,
		args:     args,
		target:   target,
		manifest: *manifest,
	}

	return pod, nil
}

func readPodManifest(manifestPath string) (*common.PodManifest, error) {
	source, err := ioutil.ReadFile(manifestPath)
	if err != nil {
		return nil, err
	}

	manifest := &common.PodManifest{}
	err = yaml.Unmarshal([]byte(source), manifest)
	if err != nil {
		return nil, err
	}

	for i, app := range manifest.Pod.Apps {
		if app.Name == "" {
			manifest.Pod.Apps[i].Name = app.Dependencies[0].ShortName()
		}
	}
	//TODO check that there is no app name conflict
	return manifest, nil
}

func (p *Pod) toAciManifestTemplate(e common.RuntimeApp) (string, error) {
	fullname := common.NewACFullName(p.manifest.Name.Name() + "_" + e.Name + ":" + p.manifest.Name.Version())
	manifest := &common.AciManifest{
		Aci: common.AciDefinition{
			Annotations:   e.Annotations,
			App:           e.App,
			Dependencies:  e.Dependencies,
			PathWhitelist: nil, // TODO
		},
		NameAndVersion: *fullname,
	}
	content, err := yaml.Marshal(manifest)
	if err != nil {
		return "", errs.WithEF(err, p.fields, "Failed to prepare manifest template")
	}
	return string(content), nil
}
