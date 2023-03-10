package decompiler

import (
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/moby/buildkit/frontend/dockerfile/parser"
	"sort"
	"strings"
)

const NOP = "#(nop) "

const RUN_PREFIX = "/bin/sh -c "

const RUN_INSTRUCTION = "RUN "
const CMD_INSTRUCTION = "CMD "
const LABEL_INSTRUCTION = "LABEL "
const MAINTAINER_INSTRUCTION = "MAINAINER "
const EXPOSE_INSTRUCTION = "EXPOSE "
const ENV_INSTRUCTION = "ENV "
const ADD_INSTRUCTION = "ADD "
const COPY_INSTRUCTION = "COPY "
const ENTRYPOINT_INSTRUCTION = "ENTRYPOINT "
const VOLUME_INSTRUCTION = "VOLUME "
const USER_INSTRUCTION = "USER "
const WORKDIR_INSTRUCTION = "WORKDIR "
const ARG_INSTRUCTION = "ARG "
const ONBUILD_INSTRUCTION = "ONBUILD "
const STOPSIGNAL_INSTRUCTION = "STOPSIGNAL "
const HEALTHCHECK_INSTRUCTION = "HEALTHCHECK "
const SHELL_INSTRUCTION = "SHELL "

var CONTAINERFILE_INSTRUCTIONS = [...]string{
	RUN_INSTRUCTION,
	CMD_INSTRUCTION,
	LABEL_INSTRUCTION,
	MAINTAINER_INSTRUCTION,
	EXPOSE_INSTRUCTION,
	ENV_INSTRUCTION,
	ADD_INSTRUCTION,
	COPY_INSTRUCTION,
	ENTRYPOINT_INSTRUCTION,
	VOLUME_INSTRUCTION,
	USER_INSTRUCTION,
	WORKDIR_INSTRUCTION,
	ARG_INSTRUCTION,
	ONBUILD_INSTRUCTION,
	STOPSIGNAL_INSTRUCTION,
	HEALTHCHECK_INSTRUCTION,
	SHELL_INSTRUCTION}

type OrderedHistory []v1.History

func (o OrderedHistory) Len() int {
	return len(o)
}

func (o OrderedHistory) Less(i, j int) bool {
	return o[i].Created.Before(o[j].Created.Time)
}

func (o OrderedHistory) Swap(i, j int) {
	o[i], o[j] = o[j], o[i]
}

func Decompile(imageName string) (*parser.Node, error) {
	ref, err := name.ParseReference(imageName)
	if err != nil {
		panic(err)
	}

	img, err := remote.Image(ref, remote.WithAuthFromKeychain(authn.DefaultKeychain))
	if err != nil {
		panic(err)
	}

	configFile, err := img.ConfigFile()
	if err != nil {
		panic(err)
	}

	root := &parser.Node{}

	history := configFile.History
	sort.Sort(OrderedHistory(history))
	for _, hist := range history {
		if hist.Comment != "" && strings.HasPrefix(hist.Comment, "FROM ") {
			err := line2Node(hist.Comment, root)
			if err != nil {
				return nil, err
			}
		}
		if hist.CreatedBy != "" {
			cmd := extractCmd(hist.CreatedBy)
			if cmd != "" {
				err := line2Node(cmd, root)
				if err != nil {
					return nil, err
				}
			}
		}
	}

	if configFile.Config.User != "" {
		err := line2Node("USER "+configFile.Config.User, root)
		if err != nil {
			return nil, err
		}
	}

	return root, nil
}

func line2Node(line string, root *parser.Node) error {
	result, err := parser.Parse(strings.NewReader(line))
	//some LABEL instructions are wrongly encoded by image producer (eg nginx) causing parser to fail
	// so we just skip in case of error and do not report
	if err == nil {
		for _, node := range result.AST.Children {
			root.AddChild(node, -1, -1)
		}
	}
	return nil
}

func extractCmd(str string) string {
	index := strings.Index(str, NOP)
	if index > 0 {
		return str[index+len(NOP):]
	}
	index = strings.Index(str, RUN_PREFIX)
	if index >= 0 {
		return "RUN " + str[index+len(RUN_PREFIX):]
	}
	if isContainerFileInstruction(str) {
		return str
	}
	return ""
}

func isContainerFileInstruction(str string) bool {
	for _, prefix := range CONTAINERFILE_INSTRUCTIONS {
		if strings.HasPrefix(str, prefix) {
			return true
		}
	}
	return false
}
