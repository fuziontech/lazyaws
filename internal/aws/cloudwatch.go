package aws

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
)

// InstanceMetrics represents CloudWatch metrics for an EC2 instance
type InstanceMetrics struct {
	InstanceID       string
	CPUUtilization   float64
	NetworkIn        float64
	NetworkOut       float64
	DiskReadBytes    float64
	DiskWriteBytes   float64
	StatusCheckFailed int
	Period           string
}

// GetInstanceMetrics retrieves CloudWatch metrics for an EC2 instance
func (c *Client) GetInstanceMetrics(ctx context.Context, instanceID string) (*InstanceMetrics, error) {
	metrics := &InstanceMetrics{
		InstanceID: instanceID,
		Period:     "Last 5 minutes",
	}

	// Time range: last 5 minutes
	endTime := time.Now()
	startTime := endTime.Add(-5 * time.Minute)

	// Helper function to get metric statistics
	getMetric := func(metricName string, stat string) (float64, error) {
		namespace := "AWS/EC2"
		dimensionName := "InstanceId"
		input := &cloudwatch.GetMetricStatisticsInput{
			Namespace:  &namespace,
			MetricName: &metricName,
			Dimensions: []types.Dimension{
				{
					Name:  &dimensionName,
					Value: &instanceID,
				},
			},
			StartTime:  &startTime,
			EndTime:    &endTime,
			Period:     getInt32Ptr(300), // 5 minutes
			Statistics: []types.Statistic{types.Statistic(stat)},
		}

		result, err := c.CloudWatch.GetMetricStatistics(ctx, input)
		if err != nil {
			return 0, err
		}

		if len(result.Datapoints) == 0 {
			return 0, nil
		}

		// Return the most recent datapoint
		var latestDatapoint types.Datapoint
		var latestTime time.Time
		for _, dp := range result.Datapoints {
			if dp.Timestamp != nil && dp.Timestamp.After(latestTime) {
				latestTime = *dp.Timestamp
				latestDatapoint = dp
			}
		}

		switch stat {
		case "Average":
			if latestDatapoint.Average != nil {
				return *latestDatapoint.Average, nil
			}
		case "Sum":
			if latestDatapoint.Sum != nil {
				return *latestDatapoint.Sum, nil
			}
		case "Maximum":
			if latestDatapoint.Maximum != nil {
				return *latestDatapoint.Maximum, nil
			}
		}

		return 0, nil
	}

	// Get CPU utilization
	cpu, err := getMetric("CPUUtilization", "Average")
	if err != nil {
		return nil, fmt.Errorf("failed to get CPU metrics: %w", err)
	}
	metrics.CPUUtilization = cpu

	// Get network metrics
	networkIn, err := getMetric("NetworkIn", "Sum")
	if err != nil {
		return nil, fmt.Errorf("failed to get network in metrics: %w", err)
	}
	metrics.NetworkIn = networkIn

	networkOut, err := getMetric("NetworkOut", "Sum")
	if err != nil {
		return nil, fmt.Errorf("failed to get network out metrics: %w", err)
	}
	metrics.NetworkOut = networkOut

	// Get disk metrics
	diskRead, err := getMetric("DiskReadBytes", "Sum")
	if err != nil {
		return nil, fmt.Errorf("failed to get disk read metrics: %w", err)
	}
	metrics.DiskReadBytes = diskRead

	diskWrite, err := getMetric("DiskWriteBytes", "Sum")
	if err != nil {
		return nil, fmt.Errorf("failed to get disk write metrics: %w", err)
	}
	metrics.DiskWriteBytes = diskWrite

	// Get status check failed metric
	statusCheck, err := getMetric("StatusCheckFailed", "Maximum")
	if err != nil {
		return nil, fmt.Errorf("failed to get status check metrics: %w", err)
	}
	metrics.StatusCheckFailed = int(statusCheck)

	return metrics, nil
}

// Helper function to get int32 pointer
func getInt32Ptr(i int32) *int32 {
	return &i
}
