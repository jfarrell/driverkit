package builder

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"

	"github.com/falcosecurity/driverkit/pkg/modulebuilder/buildtype"

	"github.com/falcosecurity/driverkit/pkg/kernelrelease"
)

const BuildTypeUbuntuGeneric buildtype.BuildType = "ubuntu-generic"
const BuildTypeUbuntuAWS buildtype.BuildType = "ubuntu-aws"

func init() {
	buildtype.EnabledBuildTypes[BuildTypeUbuntuGeneric] = true
	buildtype.EnabledBuildTypes[BuildTypeUbuntuAWS] = true
}

// UbuntuGeneric ...
type UbuntuGeneric struct {
}

// Script ...
func (v UbuntuGeneric) Script(bc BuilderConfig) (string, error) {
	t := template.New(string(BuildTypeUbuntuGeneric))
	parsed, err := t.Parse(ubuntuTemplate)
	if err != nil {
		return "", err
	}

	kr := kernelrelease.FromString(bc.Build.KernelRelease)

	urls, err := getResolvingURLs(fetchUbuntuGenericKernelURL(kr, bc.Build.KernelVersion))
	if err != nil {
		return "", err
	}
	if len(urls) != 2 {
		return "", fmt.Errorf("specific kernel headers not found")
	}

	td := ubuntuTemplateData{
		ModuleBuildDir:       ModuleDirectory,
		ModuleDownloadURL:    fmt.Sprintf("%s/%s.tar.gz", bc.ModuleConfig.DownloadBaseURL, bc.Build.ModuleVersion),
		KernelDownloadURLS:   urls,
		KernelLocalVersion:   kr.FullExtraversion,
		KernelHeadersPattern: "*generic",
	}

	buf := bytes.NewBuffer(nil)
	err = parsed.Execute(buf, td)
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}

// UbuntuAWS ...
type UbuntuAWS struct {
}

// Script ...
func (v UbuntuAWS) Script(bc BuilderConfig) (string, error) {
	t := template.New(string(BuildTypeUbuntuGeneric))
	parsed, err := t.Parse(ubuntuTemplate)
	if err != nil {
		return "", err
	}

	kr := kernelrelease.FromString(bc.Build.KernelRelease)

	urls, err := getResolvingURLs(fetchUbuntuAWSKernelURLS(kr, bc.Build.KernelVersion))
	if err != nil {
		return "", err
	}
	if len(urls) != 2 {
		return "", fmt.Errorf("specific kernel headers not found")
	}

	td := ubuntuTemplateData{
		ModuleBuildDir:       ModuleDirectory,
		ModuleDownloadURL:    moduleDownloadURL(bc),
		KernelDownloadURLS:   urls,
		KernelLocalVersion:   kr.FullExtraversion,
		KernelHeadersPattern: "*",
	}

	buf := bytes.NewBuffer(nil)
	err = parsed.Execute(buf, td)
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}

func fetchUbuntuGenericKernelURL(kr kernelrelease.KernelRelease, kernelVersion uint16) []string {
	firstExtra := extractExtraNumber(kr.Extraversion)
	return []string{
		fmt.Sprintf(
			"https://mirrors.edge.kernel.org/ubuntu/pool/main/l/linux/linux-headers-%s-%s_%s-%s.%d_all.deb",
			kr.Fullversion,
			firstExtra,
			kr.Fullversion,
			firstExtra,
			kernelVersion,
		),
		fmt.Sprintf(
			"https://mirrors.edge.kernel.org/ubuntu/pool/main/l/linux/linux-headers-%s%s_%s-%s.%d_amd64.deb",
			kr.Fullversion,
			kr.FullExtraversion,
			kr.Fullversion,
			firstExtra,
			kernelVersion,
		),
	}
}

func fetchUbuntuAWSKernelURLS(kr kernelrelease.KernelRelease, kernelVersion uint16) []string {
	firstExtra := extractExtraNumber(kr.Extraversion)
	return []string{
		fmt.Sprintf(
			"https://mirrors.edge.kernel.org/ubuntu/pool/main/l/linux-aws/linux-aws-headers-%s-%s_%s-%s.%d_all.deb",
			kr.Fullversion,
			firstExtra,
			kr.Fullversion,
			firstExtra,
			kernelVersion,
		),
		fmt.Sprintf(
			"https://mirrors.edge.kernel.org/ubuntu/pool/main/l/linux-aws/linux-headers-%s%s_%s-%s.%d_amd64.deb",
			kr.Fullversion,
			kr.FullExtraversion,
			kr.Fullversion,
			firstExtra,
			kernelVersion,
		),
	}
}

func extractExtraNumber(extraversion string) string {
	firstExtraSplit := strings.Split(extraversion, "-")
	if len(firstExtraSplit) > 0 {
		return firstExtraSplit[0]
	}
	return ""
}

type ubuntuTemplateData struct {
	ModuleBuildDir       string
	ModuleDownloadURL    string
	KernelDownloadURLS   []string
	KernelLocalVersion   string
	KernelHeadersPattern string
}

const ubuntuTemplate = `
#!/bin/bash
set -xeuo pipefail

rm -Rf {{ .ModuleBuildDir }}
mkdir {{ .ModuleBuildDir }}
rm -Rf /tmp/module-download
mkdir -p /tmp/module-download

curl --silent -SL {{ .ModuleDownloadURL }} | tar -xzf - -C /tmp/module-download
mv /tmp/module-download/*/driver/* {{ .ModuleBuildDir }}

cp /module-builder/module-Makefile {{ .ModuleBuildDir }}/Makefile
cp /module-builder/module-driver-config.h {{ .ModuleBuildDir }}/driver_config.h

# Fetch the kernel
mkdir /tmp/kernel-download
cd /tmp/kernel-download
{{range $url := .KernelDownloadURLS}}
curl --silent -o kernel.deb -SL {{ $url }}
ar x kernel.deb
tar -xvf data.tar.xz
{{end}}
ls -la /tmp/kernel-download

cd /tmp/kernel-download/usr/src/
sourcedir=$(find . -type d -name "linux-headers{{ .KernelHeadersPattern }}" | head -n 1 | xargs readlink -f)

ls -la $sourcedir
# Prepare the kernel

cd $sourcedir
cp /module-builder/kernel.config /tmp/kernel.config

{{ if .KernelLocalVersion}}
sed -i 's/^CONFIG_LOCALVERSION=.*$/CONFIG_LOCALVERSION="{{ .KernelLocalVersion }}"/' /tmp/kernel.config
{{ end }}


# Build the module
cd {{ .ModuleBuildDir }}
make KERNELDIR=$sourcedir
# Print results
ls -la

modinfo falco.ko
`
