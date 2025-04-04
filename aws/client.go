package aws

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/sirupsen/logrus"
)

type Client struct {
	ec2Client *ec2.Client
	logger    *logrus.Logger
	region    string
}

func NewClient(region string, logger *logrus.Logger) (*Client, error) {
	if logger == nil {
		logger = logrus.New()
		logger.SetLevel(logrus.InfoLevel)
	}

	cfg, err := loadAWSConfig(region)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	return &Client{
		ec2Client: ec2.NewFromConfig(cfg),
		logger:    logger,
		region:    cfg.Region,
	}, nil
}

func loadAWSConfig(region string) (aws.Config, error) {
	ctx := context.Background()
	opts := []func(*config.LoadOptions) error{}
	
	if region != "" {
		opts = append(opts, config.WithRegion(region))
	}
	
	cfg, err := config.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return aws.Config{}, err
	}
	
	return cfg, nil
}

func (c *Client) GetEC2InstanceConfig(instanceID string) (map[string]interface{}, error) {
	ctx := context.Background()
	
	resp, err := c.ec2Client.DescribeInstances(ctx, &ec2.DescribeInstancesInput{
		InstanceIds: []string{instanceID},
	})
	
	if err != nil {
		return nil, fmt.Errorf("error describing instance %s: %w", instanceID, err)
	}
	
	if len(resp.Reservations) == 0 || len(resp.Reservations[0].Instances) == 0 {
		return nil, fmt.Errorf("instance %s not found", instanceID)
	}
	
	instance := resp.Reservations[0].Instances[0]
	return c.mapInstanceToConfig(instance)
}

func (c *Client) mapInstanceToConfig(instance types.Instance) (map[string]interface{}, error) {
	config := make(map[string]interface{})
	
	config["instance_type"] = string(instance.InstanceType)
	config["ami"] = aws.ToString(instance.ImageId)
	config["subnet_id"] = aws.ToString(instance.SubnetId)
	config["associate_public_ip_address"] = instance.PublicIpAddress != nil
	
	securityGroups := make([]string, 0, len(instance.SecurityGroups))
	for _, sg := range instance.SecurityGroups {
		securityGroups = append(securityGroups, aws.ToString(sg.GroupId))
	}
	config["vpc_security_group_ids"] = securityGroups
	
	tags := make(map[string]string)
	for _, tag := range instance.Tags {
		tags[aws.ToString(tag.Key)] = aws.ToString(tag.Value)
	}
	config["tags"] = tags

	blockDevices := make([]map[string]interface{}, 0, len(instance.BlockDeviceMappings))
	for _, bdm := range instance.BlockDeviceMappings {
		if bdm.Ebs != nil {
			device := make(map[string]interface{})
			device["device_name"] = aws.ToString(bdm.DeviceName)
			device["volume_id"] = aws.ToString(bdm.Ebs.VolumeId)
			device["delete_on_termination"] = aws.ToBool(bdm.Ebs.DeleteOnTermination)
			
			volumeInfo, err := c.getVolumeInfo(aws.ToString(bdm.Ebs.VolumeId))
			if err != nil {
				c.logger.Warnf("Failed to get volume information for %s: %v", aws.ToString(bdm.Ebs.VolumeId), err)
			} else {
				for k, v := range volumeInfo {
					device[k] = v
				}
			}
			
			blockDevices = append(blockDevices, device)
		}
	}
	config["ebs_block_device"] = blockDevices
	
	return config, nil
}

func (c *Client) getVolumeInfo(volumeID string) (map[string]interface{}, error) {
	ctx := context.Background()
	
	resp, err := c.ec2Client.DescribeVolumes(ctx, &ec2.DescribeVolumesInput{
		VolumeIds: []string{volumeID},
	})
	
	if err != nil {
		return nil, fmt.Errorf("error describing volume %s: %w", volumeID, err)
	}
	
	if len(resp.Volumes) == 0 {
		return nil, fmt.Errorf("volume %s not found", volumeID)
	}
	
	volume := resp.Volumes[0]
	volumeInfo := make(map[string]interface{})
	
	volumeInfo["volume_size"] = volume.Size
	volumeInfo["volume_type"] = string(volume.VolumeType)
	volumeInfo["encrypted"] = volume.Encrypted
	
	if volume.Iops != nil {
		volumeInfo["iops"] = volume.Iops
	}
	
	return volumeInfo, nil
}