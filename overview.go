package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/aws/aws-sdk-go/service/s3"
)

type Overview struct {
	Instances *ec2.DescribeInstancesOutput
	SecurityGroups *ec2.DescribeSecurityGroupsOutput
	Vpcs *ec2.DescribeVpcsOutput
	Subnets *ec2.DescribeSubnetsOutput
	RouteTables *ec2.DescribeRouteTablesOutput
	VpcPeeringConnections *ec2.DescribeVpcPeeringConnectionsOutput
	Buckets *s3.ListBucketsOutput
	DBInstances *rds.DescribeDBInstancesOutput
}

func processOverview(config Config, attachment *Attachment) {
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
	instanceCount := 0
	instanceMap := make(map[string]ec2.Instance)
	for _, reservation := range instances.Reservations {
		for _, instance := range reservation.Instances {
			fmt.Println("-", *instance.InstanceId)
			instanceMap[*instance.InstanceId] = *instance
			if *instance.State.Name == "running" {
				instanceCount++
			}
		}
	}
	attachment.Fields = append(attachment.Fields, Field{Title: "Running instances", Value: fmt.Sprintf("%d", instanceCount), Short: true})

	securityGroups, err := ec2client.DescribeSecurityGroups(nil)
	if err != nil {
		panic(err)
	}

	securityGroupMap := make(map[string]ec2.SecurityGroup)
	for _, securityGroup := range securityGroups.SecurityGroups {
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

	s3client := s3.New(&aws.Config{Region: aws.String(config.General.DefaultRegion)})

	buckets, err := s3client.ListBuckets(nil)
	if err != nil {
		panic(err)
	}
	attachment.Fields = append(attachment.Fields, Field{Title: "S3 buckets", Value: fmt.Sprintf("%d", len(buckets.Buckets)), Short: true})

	rdsclient := rds.New(&aws.Config{Region: aws.String(config.General.DefaultRegion)})

	dbInstances, err := rdsclient.DescribeDBInstances(nil)
	if err != nil {
		panic(err)
	}
	attachment.Fields = append(attachment.Fields, Field{Title: "DB instances", Value: fmt.Sprintf("%d", len(dbInstances.DBInstances)), Short: true})

	overview := Overview{Instances: instances, SecurityGroups: securityGroups, Vpcs: vpcs, Subnets: subnets, RouteTables: routeTables, VpcPeeringConnections: vpcPeeringConnections, Buckets: buckets, DBInstances: dbInstances}
	jsonData, err := json.MarshalIndent(overview, "", "  ")
	if err != nil {
		panic(err)
	}

	const outputDir = "data/aws-dashboard"
	err = os.MkdirAll(outputDir, os.ModeDir)
	if err != nil {
		panic(err)
	}

	err = ioutil.WriteFile(outputDir + "/aws.json", jsonData, 0777)
	if err != nil {
		panic(err)
	}

	var m json.RawMessage
	err = m.UnmarshalJSON(jsonData)
	if err != nil {
		panic(err)
	}
}
