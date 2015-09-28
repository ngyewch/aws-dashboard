package main

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
)

func main() {
	ec2client := ec2.New(&aws.Config{Region: aws.String("ap-southeast-1")})
	regions, err := ec2client.DescribeRegions(&ec2.DescribeRegionsInput{})
	if err != nil {
		panic(err)
	}
	fmt.Println("Regions:")
	for _, region := range regions.Regions {
		fmt.Println("- ", *region.RegionName)
	}

	svc := ec2.New(&aws.Config{Region: aws.String("ap-southeast-1")})

	instances, err := svc.DescribeInstances(nil)
	if err != nil {
		panic(err)
	}

	fmt.Println("Instances:")
	instanceMap := make(map[string]ec2.Instance)
	for _, reservation := range instances.Reservations {
		for _, instance := range reservation.Instances {
			fmt.Println("- ", *instance.InstanceId)
			instanceMap[*instance.InstanceId] = *instance
		}
	}

	securityGroups, err := svc.DescribeSecurityGroups(nil)
	if err != nil {
		panic(err)
	}

	fmt.Println("Security Groups:")
	securityGroupMap := make(map[string]ec2.SecurityGroup)
	for _, securityGroup := range securityGroups.SecurityGroups {
		fmt.Println("- ", *securityGroup.GroupId, *securityGroup.GroupName)
		securityGroupMap[*securityGroup.GroupId] = *securityGroup
	}
}
