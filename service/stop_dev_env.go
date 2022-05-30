package service

import (
	"encoding/json"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/recode-sh/aws-cloud-provider/infrastructure"
	"github.com/recode-sh/recode/entities"
	"github.com/recode-sh/recode/stepper"
)

func (a *AWS) StopDevEnv(
	stepper stepper.Stepper,
	config *entities.Config,
	cluster *entities.Cluster,
	devEnv *entities.DevEnv,
) error {

	var devEnvInfra *DevEnvInfrastructure
	err := json.Unmarshal([]byte(devEnv.InfrastructureJSON), &devEnvInfra)

	if err != nil {
		return err
	}

	ec2Client := ec2.NewFromConfig(a.sdkConfig)

	stepper.StartTemporaryStep("Waiting for the EC2 instance to stop")

	return infrastructure.StopInstance(
		ec2Client,
		devEnvInfra.Instance,
	)
}
