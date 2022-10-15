package main

import (
	"fmt"

	"github.com/pulumi/pulumi-aws/sdk/v5/go/aws/ec2"
	"github.com/pulumi/pulumi-aws/sdk/v5/go/aws/rds"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

type Configs struct {
	VpcNetwork string
	Active     bool
	Subnets    map[string]string
	Database   map[string]interface{}
}

func main() {

	pulumi.Run(func(ctx *pulumi.Context) error {

		// get configs
		var data Configs
		cfg := config.New(ctx, "")
		cfg.RequireObject("dev", &data)

		// create vpc
		vpc, err := ec2.NewVpc(ctx, "vpc", &ec2.VpcArgs{
			CidrBlock: pulumi.String("10.0.0.0/16"),
			Tags: pulumi.StringMap{
				"Name": pulumi.String("vpc-eu-central-1"),
			},
		})
		if err != nil {
			return err
		}

		ctx.Export("VPC-ID", vpc.ID())

		// create subnets
		var subnetGroups []pulumi.StringInput
		for az, subnet := range data.Subnets {

			fmt.Println("the subnet is:", subnet)

			subnetGroupEUCentral, err := ec2.NewSubnet(ctx, fmt.Sprintf("db-subnet-%s", az), &ec2.SubnetArgs{
				VpcId:            vpc.ID(),
				AvailabilityZone: pulumi.String(az),
				CidrBlock:        pulumi.String(subnet),
				Tags: pulumi.StringMap{
					"Name": pulumi.String(fmt.Sprintf("db-subnet-%s\n", az)),
				},
			})
			if err != nil {
				return err
			}
			ctx.Export(fmt.Sprintf("db-subnet-%s", az), subnetGroupEUCentral)
			subnetGroups = append(subnetGroups, subnetGroupEUCentral.ID())

		}

		// create db subnet group
		dbSubnetGroup, err := rds.NewSubnetGroup(ctx, "pulumi-db-subnet-group", &rds.SubnetGroupArgs{
			SubnetIds: pulumi.StringArray{
				// Todo: will remove this hardcoding once I figure out how to append to a slice and convert it to pulumi string array
				subnetGroups[0],
				subnetGroups[1],
				subnetGroups[2],
			},
			Tags: pulumi.StringMap{
				"Name": pulumi.String("pulumi-db-subnet-group"),
			},
		})
		if err != nil {
			return err
		}

		ctx.Export("DB-Subnet-Group", dbSubnetGroup.Name)

		// create parameter group
		dbParameterGroup, err := rds.NewParameterGroup(ctx, "pulumi-managed-parameter-group", &rds.ParameterGroupArgs{
			Family: pulumi.String("mysql5.7"),
			Parameters: rds.ParameterGroupParameterArray{
				&rds.ParameterGroupParameterArgs{
					Name:  pulumi.String("character_set_server"),
					Value: pulumi.String("utf8"),
				},
				&rds.ParameterGroupParameterArgs{
					Name:  pulumi.String("max_connections"),
					Value: pulumi.String(data.Database["max_connections"].(string)),
				},
			},
		})
		if err != nil {
			return err
		}

		ctx.Export("DB Instance Password", dbParameterGroup.Name)

		// create rds
		dbInstance, err := rds.NewInstance(ctx, "pulumi-rds-instance", &rds.InstanceArgs{
			AllocatedStorage:   pulumi.Int(data.Database["allocated_storage"].(float64)),
			Engine:             pulumi.String(data.Database["engine"].(string)),
			EngineVersion:      pulumi.String(data.Database["engine_version"].(string)),
			InstanceClass:      pulumi.String(data.Database["instance_size"].(string)),
			DbSubnetGroupName:  dbSubnetGroup.Name,
			ParameterGroupName: dbParameterGroup.Name,
			// fingure out use of encrypted password later
			Password:          pulumi.String(data.Database["password"].(string)),
			SkipFinalSnapshot: pulumi.Bool(data.Database["skip_final_snapshot"].(bool)),
			// username mysql is not accepted, put a check on it
			Username: pulumi.String(data.Database["username"].(string)),
		})
		if err != nil {
			return err
		}
		ctx.Export("DB Instance Password", dbInstance.Endpoint)
		ctx.Export("DB Instance Password", dbInstance.Password)
		return nil
	})
}
