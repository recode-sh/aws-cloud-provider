package service

import (
	"encoding/json"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/recode-sh/agent/constants"
	"github.com/recode-sh/aws-cloud-provider/infrastructure"
	"github.com/recode-sh/recode/entities"
	"github.com/recode-sh/recode/stepper"
)

func (a *AWS) StartDevEnv(
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

	stepper.StartTemporaryStep("Starting the EC2 instance")

	ec2Client := ec2.NewFromConfig(a.sdkConfig)
	err = infrastructure.StartInstance(
		ec2Client,
		devEnvInfra.Instance,
	)

	if err != nil {
		return err
	}

	stepper.StartTemporaryStep("Waiting for SSH to be available in the EC2 instance")

	err = infrastructure.WaitForSSHAvailableInInstance(
		ec2Client,
		devEnvInfra.Instance.PublicIPAddress,
		constants.SSHServerListenPort,
	)

	if err != nil {
		return err
	}

	devEnv.SetInfrastructureJSON(devEnvInfra)

	devEnv.InstancePublicIPAddress = devEnvInfra.Instance.PublicIPAddress
	devEnv.InstancePublicHostname = devEnvInfra.Instance.PublicHostname

	return nil
}
