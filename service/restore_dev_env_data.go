package service

import (
	"encoding/json"
	"sync"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/recode-sh/aws-cloud-provider/infrastructure"
	"github.com/recode-sh/recode/entities"
)

func (a *AWS) RestoreDevEnvData(
	config *entities.Config,
	cluster *entities.Cluster,
	devEnv *entities.DevEnv,
) (*string, error) {

	var clusterInfra *ClusterInfrastructure
	err := json.Unmarshal([]byte(cluster.InfrastructureJSON), &clusterInfra)

	if err != nil {
		return nil, err
	}

	var devEnvInfra *DevEnvInfrastructure
	err = json.Unmarshal([]byte(devEnv.InfrastructureJSON), &devEnvInfra)

	if err != nil {
		return nil, err
	}

	ec2Client := ec2.NewFromConfig(a.sdkConfig)
	prefixResource := prefixDevEnvResource(cluster.GetNameSlug(), devEnv.GetNameSlug())

	var attacheVolumeWG sync.WaitGroup
	attachVolumeErrors := make([]error, len(devEnvInfra.Instance.Volumes))

	devEnvInfraUpdatedVolumes := make([]infrastructure.InstanceVolume, len(devEnvInfra.Instance.Volumes))

	for i, volume := range devEnvInfra.Instance.Volumes {
		attacheVolumeWG.Add(1)

		go func(i int, volume infrastructure.InstanceVolume) {
			defer attacheVolumeWG.Done()

			volumeName := "root-volume"

			createVolumeResp := infrastructure.CreateVolumeFromSnapshot(
				ec2Client,
				prefixResource(volumeName),
				clusterInfra.Subnet.AvailabilityZone,
				volume.SnapshotID,
			)

			if createVolumeResp.Err != nil {
				attachVolumeErrors[i] = createVolumeResp.Err
				return
			}

			devEnvInfraUpdatedVolumes[i] = volume
			devEnvInfraUpdatedVolumes[i].ID = createVolumeResp.VolumeID

			attachVolumeResp := infrastructure.AttachVolume(
				ec2Client,
				devEnvInfra.Instance.ID,
				createVolumeResp.VolumeID,
				volume.DeviceName,
			)

			if attachVolumeResp.Err != nil {
				attachVolumeErrors[i] = attachVolumeResp.Err
				return
			}
		}(i, volume)
	}

	attacheVolumeWG.Wait()

	for _, err := range attachVolumeErrors {
		if err == nil {
			continue
		}

		return nil, err
	}

	devEnvInfra.Instance.Volumes = devEnvInfraUpdatedVolumes
	devEnvInfraJSON, err := json.Marshal(devEnvInfra)

	if err != nil {
		return nil, err
	}

	s := string(devEnvInfraJSON)

	return &s, nil
}
