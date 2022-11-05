// Copyright © 2021 Alibaba Group Holding Ltd.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package guest

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/labring/sealos/pkg/ssh"

	"github.com/labring/sealos/pkg/constants"
	fileutil "github.com/labring/sealos/pkg/utils/file"
	"github.com/labring/sealos/pkg/utils/maps"

	"github.com/pkg/errors"
	"k8s.io/client-go/util/homedir"

	"github.com/labring/sealos/fork/golang/expansion"
	"github.com/labring/sealos/pkg/env"
	"github.com/labring/sealos/pkg/image"
	"github.com/labring/sealos/pkg/image/types"
	"github.com/labring/sealos/pkg/runtime"
	v2 "github.com/labring/sealos/pkg/types/v1beta1"
)

type Interface interface {
	Apply(cluster *v2.Cluster, mounts []v2.MountImage) error
	Delete(cluster *v2.Cluster) error
}

type Default struct {
	imageService types.ImageService
	ssh          ssh.Interface
}

func NewGuestManager() (Interface, error) {
	is, err := image.NewImageService()
	if err != nil {
		return nil, err
	}
	return &Default{imageService: is}, nil
}

func (d *Default) Apply(cluster *v2.Cluster, mounts []v2.MountImage) error {
	envInterface := env.NewEnvProcessor(cluster, cluster.Status.Mounts)
	envs := envInterface.WrapperEnv(cluster.GetMaster0IP()) //clusterfile

	kubeConfig := filepath.Join(homedir.HomeDir(), ".kube", "config")
	if !fileutil.IsExist(kubeConfig) {
		adminFile := runtime.GetConstantData(cluster.Name).AdminFile()
		data, err := fileutil.ReadAll(adminFile)
		if err != nil {
			return errors.Wrap(err, "read admin.conf error in guest")
		}
		master0IP := cluster.GetMaster0IP()
		outData := strings.ReplaceAll(string(data), runtime.DefaultAPIServerDomain, master0IP)
		if err = fileutil.WriteFile(kubeConfig, []byte(outData)); err != nil {
			return err
		}
		defer func() {
			_ = fileutil.CleanFiles(kubeConfig)
		}()
	}
	d.ssh = ssh.NewSSHClient(&cluster.Spec.SSH, true)

	return d.getGuestCmd(envs, cluster, mounts)
}

// run command
func (d *Default) runCmd(cluster *v2.Cluster, name, cmd string) error {
	if cmd == "" {
		return nil
	}

	if err := d.ssh.CmdAsync(cluster.GetMaster0IPAndPort(),
		fmt.Sprintf(constants.CdAndExecCmd, constants.GetAppWorkDir(cluster.Name, name),
			cmd)); err != nil {
		return err
	}
	return nil
}

func (d *Default) getGuestCmd(envs map[string]string, cluster *v2.Cluster, mounts []v2.MountImage) error {
	overrideCmd := cluster.Spec.Command

	for idx, i := range mounts {
		mergeENV := maps.MergeMap(i.Env, envs)
		mapping := expansion.MappingFuncFor(mergeENV)
		for _, cmd := range i.Entrypoint {
			if err := d.runCmd(cluster, i.Name, expansion.Expand(cmd, mapping)); err != nil {
				return fmt.Errorf("run entrypoint command %s error: %v", cmd, err)
			}
		}

		// if --cmd is specified, only the CMD of the first MountImage will be overridden
		if idx == 0 && len(overrideCmd) > 0 {
			for _, cmd := range overrideCmd {
				if err := d.runCmd(cluster, i.Name, expansion.Expand(cmd, mapping)); err != nil {
					return fmt.Errorf("run override command %s error: %v", cmd, err)
				}
			}
			continue
		}

		for _, cmd := range i.Cmd {
			if err := d.runCmd(cluster, i.Name, expansion.Expand(cmd, mapping)); err != nil {
				return fmt.Errorf("run cmd command %s error: %v", cmd, err)
			}
		}
	}

	return nil
}

func (d Default) Delete(cluster *v2.Cluster) error {
	panic("implement me")
}
