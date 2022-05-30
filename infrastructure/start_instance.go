package infrastructure

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

type StartInstanceResp struct {
	Err      error
	Instance types.Instance
}

func StartInstance(
	ec2Client *ec2.Client,
	instance *Instance,
) error {

	_, err := ec2Client.StartInstances(context.TODO(), &ec2.StartInstancesInput{
		InstanceIds: []string{instance.ID},
	})

	if err != nil {
		return err
	}

	runningWaiter := ec2.NewInstanceRunningWaiter(ec2Client)
	maxWaitTime := 5 * time.Minute

	err = runningWaiter.Wait(
		context.TODO(),
		&ec2.DescribeInstancesInput{
			InstanceIds: []string{
				instance.ID,
			},
		},
		maxWaitTime,
	)

	if err != nil {
		return err
	}

	startedInstance, err := lookupInstance(ec2Client, instance.ID)

	if err != nil {
		return err
	}

	instance.PublicIPAddress = *startedInstance.PublicIpAddress
	instance.PublicHostname = *startedInstance.PublicDnsName

	return nil
}
