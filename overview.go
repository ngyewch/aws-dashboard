package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
)

type Overview struct {
	Instances *ec2.DescribeInstancesOutput
	SecurityGroups *ec2.DescribeSecurityGroupsOutput
	Vpcs *ec2.DescribeVpcsOutput
	Subnets *ec2.DescribeSubnetsOutput
	RouteTables *ec2.DescribeRouteTablesOutput
	VpcPeeringConnections *ec2.DescribeVpcPeeringConnectionsOutput
}

func processOverview(config Config) {
	ec2client := ec2.New(&aws.Config{Region: aws.String(config.General.DefaultRegion)})
	regions, err := ec2client.DescribeRegions(&ec2.DescribeRegionsInput{})
	if err != nil {
		panic(err)
	}
	fmt.Println("Regions:")
	for _, region := range regions.Regions {
		fmt.Println("-", *region.RegionName)
	}

	ec2client = ec2.New(&aws.Config{Region: aws.String(config.General.DefaultRegion)})

	instances, err := ec2client.DescribeInstances(nil)
	if err != nil {
		panic(err)
	}

	fmt.Println("Instances:")
	instanceMap := make(map[string]ec2.Instance)
	for _, reservation := range instances.Reservations {
		for _, instance := range reservation.Instances {
			fmt.Println("-", *instance.InstanceId)
			instanceMap[*instance.InstanceId] = *instance
		}
	}

	securityGroups, err := ec2client.DescribeSecurityGroups(nil)
	if err != nil {
		panic(err)
	}

	fmt.Println("Security Groups:")
	securityGroupMap := make(map[string]ec2.SecurityGroup)
	for _, securityGroup := range securityGroups.SecurityGroups {
		fmt.Println("-", *securityGroup.GroupId, *securityGroup.GroupName)
		securityGroupMap[*securityGroup.GroupId] = *securityGroup
	}

	vpcs, err := ec2client.DescribeVpcs(nil)
	if err != nil {
		panic(err)
	}

	subnets, err := ec2client.DescribeSubnets(nil)
	if err != nil {
		panic(err)
	}

	routeTables, err := ec2client.DescribeRouteTables(nil)
	if err != nil {
		panic(err)
	}

	vpcPeeringConnections, err := ec2client.DescribeVpcPeeringConnections(nil)
	if err != nil {
		panic(err)
	}

	overview := Overview{Instances: instances, SecurityGroups: securityGroups, Vpcs: vpcs, Subnets: subnets, RouteTables: routeTables, VpcPeeringConnections: vpcPeeringConnections}
	jsonData, err := json.MarshalIndent(overview, "", "  ")
	if err != nil {
		panic(err)
	}
	err = ioutil.WriteFile("ec2.json", jsonData, 0777)
	if err != nil {
		panic(err)
	}

	var m json.RawMessage
	err = m.UnmarshalJSON(jsonData)
	if err != nil {
		panic(err)
	}
}
