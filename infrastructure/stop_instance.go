package infrastructure

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
)

func StopInstance(
	ec2Client *ec2.Client,
	instance *Instance,
) error {

	_, err := ec2Client.StopInstances(context.TODO(), &ec2.StopInstancesInput{
		InstanceIds: []string{instance.ID},
	})

	if err != nil {
		return err
	}

	stoppedWaiter := ec2.NewInstanceStoppedWaiter(ec2Client)
	maxWaitTime := 5 * time.Minute

	return stoppedWaiter.Wait(
		context.TODO(),
		&ec2.DescribeInstancesInput{
			InstanceIds: []string{
				instance.ID,
			},
		},
		maxWaitTime,
	)
}
