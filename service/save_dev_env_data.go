package service

import (
	"encoding/json"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/recode-sh/aws-cloud-provider/infrastructure"
	"github.com/recode-sh/recode/entities"
)

func SaveDevEnvData(
	config *entities.Config,
	cluster *entities.Cluster,
	devEnv *entities.DevEnv,
) (*string, error) {

	var devEnvInfra *DevEnvInfrastructure
	err := json.Unmarshal([]byte(devEnv.InfrastructureJSON), &devEnvInfra)

	if err != nil {
		return nil, err
	}

	ec2Client := ec2.NewFromConfig(*aws.NewConfig())
	prefixResource := prefixDevEnvResource(cluster.GetNameSlug(), devEnv.GetNameSlug())

	var createSnapshotWG sync.WaitGroup
	createSnapshotErrors := make([]error, len(devEnvInfra.Instance.Volumes))

	devEnvInfraUpdatedVolumes := make([]infrastructure.InstanceVolume, len(devEnvInfra.Instance.Volumes))

	for i, volume := range devEnvInfra.Instance.Volumes {
		createSnapshotWG.Add(1)

		go func(i int, volume infrastructure.InstanceVolume) {
			defer createSnapshotWG.Done()

			snapshotName := "root-volume-snapshot"

			createSnapshotForVolumeResp := infrastructure.CreateSnapshotForVolume(
				ec2Client,
				prefixResource(snapshotName),
				volume.ID,
			)

			if createSnapshotForVolumeResp.Err != nil {
				createSnapshotErrors[i] = createSnapshotForVolumeResp.Err
				return
			}

			if len(volume.SnapshotID) > 0 { // Volume has old snapshot
				removeVolumeSnapshotResp := infrastructure.RemoveVolumeSnapshot(ec2Client, volume.SnapshotID)

				if removeVolumeSnapshotResp.Err != nil {
					createSnapshotErrors[i] = removeVolumeSnapshotResp.Err
					return
				}
			}

			devEnvInfraUpdatedVolumes[i] = volume
			devEnvInfraUpdatedVolumes[i].SnapshotID = createSnapshotForVolumeResp.SnapshotID

			detachVolumeResp := infrastructure.DetachVolume(
				ec2Client,
				devEnvInfra.Instance.ID,
				volume.ID,
				volume.DeviceName,
			)

			if detachVolumeResp.Err != nil {
				createSnapshotErrors[i] = detachVolumeResp.Err
				return
			}

			removeVolumeResp := infrastructure.RemoveVolume(
				ec2Client,
				volume.ID,
			)

			if removeVolumeResp.Err != nil {
				createSnapshotErrors[i] = removeVolumeResp.Err
				return
			}
		}(i, volume)
	}

	createSnapshotWG.Wait()

	for _, err := range createSnapshotErrors {
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
