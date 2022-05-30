package service

import (
	"encoding/json"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/recode-sh/aws-cloud-provider/infrastructure"
	"github.com/recode-sh/recode/entities"
	"github.com/recode-sh/recode/queues"
	"github.com/recode-sh/recode/stepper"
)

func (a *AWS) RemoveDevEnv(
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
	devEnvInfraQueue := queues.InfrastructureQueue[*DevEnvInfrastructure]{}

	terminateInstance := func(infra *DevEnvInfrastructure) error {
		if infra.Instance == nil {
			return nil
		}

		err := infrastructure.TerminateInstance(
			ec2Client,
			infra.Instance.ID,
		)

		if err != nil {
			return err
		}

		infra.Instance = nil
		return nil
	}

	devEnvInfraQueue = append(
		devEnvInfraQueue,
		queues.InfrastructureQueueSteps[*DevEnvInfrastructure]{
			func(*DevEnvInfrastructure) error {
				stepper.StartTemporaryStep("Waiting for the EC2 instance to terminate")
				return nil
			},
			terminateInstance,
		},
	)

	removeKeyPair := func(infra *DevEnvInfrastructure) error {
		if infra.KeyPair == nil {
			return nil
		}

		err := infrastructure.RemoveKeyPair(
			ec2Client,
			infra.KeyPair.ID,
		)

		if err != nil {
			return err
		}

		infra.KeyPair = nil
		return nil
	}

	removeNetworkInterface := func(infra *DevEnvInfrastructure) error {
		if infra.NetworkInterface == nil {
			return nil
		}

		err := infrastructure.RemoveNetworkInterface(
			ec2Client,
			infra.NetworkInterface.ID,
		)

		if err != nil {
			return err
		}

		infra.NetworkInterface = nil
		return nil
	}

	devEnvInfraQueue = append(
		devEnvInfraQueue,
		queues.InfrastructureQueueSteps[*DevEnvInfrastructure]{
			func(*DevEnvInfrastructure) error {
				stepper.StartTemporaryStep("Removing the key pair and the network interface")
				return nil
			},
			removeKeyPair,
			removeNetworkInterface,
		},
	)

	removeSecurityGroup := func(infra *DevEnvInfrastructure) error {
		if infra.SecurityGroup == nil {
			return nil
		}

		err := infrastructure.RemoveSecurityGroup(
			ec2Client,
			infra.SecurityGroup.ID,
		)

		if err != nil {
			return err
		}

		infra.SecurityGroup = nil
		return nil
	}

	devEnvInfraQueue = append(
		devEnvInfraQueue,
		queues.InfrastructureQueueSteps[*DevEnvInfrastructure]{
			func(*DevEnvInfrastructure) error {
				stepper.StartTemporaryStep("Removing the security group")
				return nil
			},
			removeSecurityGroup,
		},
	)

	err = devEnvInfraQueue.Run(
		devEnvInfra,
	)

	// Dev env infra could be updated in the queue even
	// in case of error (partial infrastructure)
	devEnv.SetInfrastructureJSON(devEnvInfra)

	return err
}
