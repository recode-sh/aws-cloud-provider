package service

import (
	"encoding/json"
	"strconv"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/recode-sh/agent/constants"
	"github.com/recode-sh/aws-cloud-provider/infrastructure"
	"github.com/recode-sh/recode/entities"
	"github.com/recode-sh/recode/queues"
	"github.com/recode-sh/recode/stepper"
)

type DevEnvInfrastructure struct {
	SecurityGroup     *infrastructure.SecurityGroup     `json:"security_group"`
	KeyPair           *infrastructure.KeyPair           `json:"key_pair"`
	NetworkInterface  *infrastructure.NetworkInterface  `json:"network_interface"`
	InstanceTypeInfos *infrastructure.InstanceTypeInfos `json:"instance_type_infos"`
	InstanceAMI       *infrastructure.AMI               `json:"instance_ami"`
	Instance          *infrastructure.Instance          `json:"instance"`
}

func (a *AWS) CreateDevEnv(
	stepper stepper.Stepper,
	config *entities.Config,
	cluster *entities.Cluster,
	devEnv *entities.DevEnv,
) error {

	var clusterInfra *ClusterInfrastructure
	err := json.Unmarshal([]byte(cluster.InfrastructureJSON), &clusterInfra)

	if err != nil {
		return err
	}

	devEnvInfra := &DevEnvInfrastructure{}
	if len(devEnv.InfrastructureJSON) > 0 {
		err := json.Unmarshal([]byte(devEnv.InfrastructureJSON), devEnvInfra)

		if err != nil {
			return err
		}
	}

	prefixResource := prefixDevEnvResource(cluster.GetNameSlug(), devEnv.GetNameSlug())
	ec2Client := ec2.NewFromConfig(a.sdkConfig)

	devEnvInfraQueue := queues.InfrastructureQueue[*DevEnvInfrastructure]{}

	createSecurityGroup := func(infra *DevEnvInfrastructure) error {
		if infra.SecurityGroup != nil {
			return nil
		}

		recodeSSHServerListenPort, err := strconv.ParseInt(
			constants.SSHServerListenPort,
			10,
			64,
		)

		if err != nil {
			return err
		}

		securityGroup, err := infrastructure.CreateSecurityGroup(
			ec2Client,
			prefixResource("security-group"),
			"The security group attached to your development environment",
			clusterInfra.VPC.ID,
			[]types.IpPermission{
				{
					IpProtocol: aws.String("tcp"),
					FromPort:   aws.Int32(int32(recodeSSHServerListenPort)),
					ToPort:     aws.Int32(int32(recodeSSHServerListenPort)),
					IpRanges: []types.IpRange{
						{
							CidrIp: aws.String("0.0.0.0/0"),
						},
					},
				},
			},
		)

		if err != nil {
			return err
		}

		infra.SecurityGroup = securityGroup
		return nil
	}

	createKeyPair := func(infra *DevEnvInfrastructure) error {
		if infra.KeyPair != nil {
			return nil
		}

		keyPair, err := infrastructure.CreateKeyPair(
			ec2Client,
			prefixResource("key-pair"),
		)

		if err != nil {
			return err
		}

		infra.KeyPair = keyPair
		return nil
	}

	devEnvInfraQueue = append(
		devEnvInfraQueue,
		queues.InfrastructureQueueSteps[*DevEnvInfrastructure]{
			func(*DevEnvInfrastructure) error {
				stepper.StartTemporaryStep("Creating a security group and a key pair")
				return nil
			},
			createSecurityGroup,
			createKeyPair,
		},
	)

	createNetworkInterface := func(infra *DevEnvInfrastructure) error {
		if infra.NetworkInterface != nil {
			return nil
		}

		networkInterface, err := infrastructure.CreateNetworkInterface(
			ec2Client,
			prefixResource("network-interface"),
			"The network interface attached to your development environment",
			clusterInfra.Subnet.ID,
			[]string{infra.SecurityGroup.ID},
		)

		if err != nil {
			return err
		}

		infra.NetworkInterface = networkInterface
		return nil
	}

	devEnvInfraQueue = append(
		devEnvInfraQueue,
		queues.InfrastructureQueueSteps[*DevEnvInfrastructure]{
			func(*DevEnvInfrastructure) error {
				stepper.StartTemporaryStep("Creating a network interface")
				return nil
			},
			createNetworkInterface,
		},
	)

	lookupInstanceTypeInfos := func(infra *DevEnvInfrastructure) error {
		if infra.InstanceTypeInfos != nil {
			return nil
		}

		instanceTypeInfos, err := infrastructure.LookupInstanceTypeInfos(
			ec2Client,
			devEnv.InstanceType,
		)

		if err != nil {
			return err
		}

		infra.InstanceTypeInfos = instanceTypeInfos
		return nil
	}

	devEnvInfraQueue = append(
		devEnvInfraQueue,
		queues.InfrastructureQueueSteps[*DevEnvInfrastructure]{
			func(*DevEnvInfrastructure) error {
				stepper.StartTemporaryStep("Looking up instance type informations")
				return nil
			},
			lookupInstanceTypeInfos,
		},
	)

	lookupUbuntuAMIForArchAndRegion := func(infra *DevEnvInfrastructure) error {
		if infra.InstanceAMI != nil {
			return nil
		}

		instanceAMI, err := infrastructure.LookupUbuntuAMIForArch(
			ec2Client,
			infra.InstanceTypeInfos.Arch,
		)

		if err != nil {
			return err
		}

		infra.InstanceAMI = instanceAMI
		return nil
	}

	devEnvInfraQueue = append(
		devEnvInfraQueue,
		queues.InfrastructureQueueSteps[*DevEnvInfrastructure]{
			func(*DevEnvInfrastructure) error {
				stepper.StartTemporaryStep("Looking up the AMI details")
				return nil
			},
			lookupUbuntuAMIForArchAndRegion,
		},
	)

	createInstance := func(infra *DevEnvInfrastructure) error {
		if infra.Instance != nil {
			return nil
		}

		instance, err := infrastructure.CreateInstance(
			ec2Client,
			prefixResource("instance"),
			infra.InstanceAMI.ID,
			infra.InstanceAMI.RootDeviceName,
			infra.InstanceTypeInfos.Type,
			infra.NetworkInterface.ID,
			infra.KeyPair.Name,
		)

		if err != nil {
			return err
		}

		infra.Instance = instance
		return nil
	}

	devEnvInfraQueue = append(
		devEnvInfraQueue,
		queues.InfrastructureQueueSteps[*DevEnvInfrastructure]{
			func(*DevEnvInfrastructure) error {
				stepper.StartTemporaryStep("Creating an EC2 instance")
				return nil
			},
			createInstance,
		},
	)

	lookupInstanceInitScriptResults := func(infra *DevEnvInfrastructure) error {
		if infra.Instance.InitScriptResults != nil {
			return nil
		}

		initScriptResults, err := infrastructure.LookupInitInstanceScriptResults(
			ec2Client,
			devEnvInfra.Instance.PublicIPAddress,
			constants.SSHServerListenPort,
			entities.DevEnvRootUser,
			devEnvInfra.KeyPair.PEMContent,
		)

		if err != nil {
			return err
		}

		infra.Instance.InitScriptResults = initScriptResults
		return nil
	}

	devEnvInfraQueue = append(
		devEnvInfraQueue,
		queues.InfrastructureQueueSteps[*DevEnvInfrastructure]{
			func(*DevEnvInfrastructure) error {
				stepper.StartTemporaryStep("Waiting for the EC2 instance to start")
				return nil
			},
			lookupInstanceInitScriptResults,
		},
	)

	err = devEnvInfraQueue.Run(devEnvInfra)

	// Dev env infra could be updated in the queue even
	// in case of error (partial infrastructure)
	devEnv.SetInfrastructureJSON(devEnvInfra)

	if err != nil {
		return err
	}

	devEnv.InstancePublicIPAddress = devEnvInfra.Instance.PublicIPAddress
	devEnv.InstancePublicHostname = devEnvInfra.Instance.PublicHostname

	devEnv.SSHHostKeys = devEnvInfra.Instance.InitScriptResults.SSHHostKeys
	devEnv.SSHKeyPairPEMContent = devEnvInfra.KeyPair.PEMContent

	return nil
}
