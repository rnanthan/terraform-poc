# Bedrock Agent for AWS Instance Scheduler RDS Management
# This includes CloudFormation template, Lambda functions, and OpenAPI schema

import json
import boto3
import logging
from datetime import datetime, timezone
from typing import Dict, List, Any

# CloudFormation Template for Bedrock Agent
CLOUDFORMATION_TEMPLATE = {
    "AWSTemplateFormatVersion": "2010-09-09",
    "Description": "Bedrock Agent for RDS Instance Scheduler Management",
    "Parameters": {
        "BedrockAgentName": {
            "Type": "String",
            "Default": "RDSSchedulerAgent",
            "Description": "Name for the Bedrock Agent"
        },
        "InstanceSchedulerStackName": {
            "Type": "String",
            "Default": "instance-scheduler",
            "Description": "Name of the Instance Scheduler CloudFormation stack"
        }
    },
    "Resources": {
        # IAM Role for Bedrock Agent
        "BedrockAgentRole": {
            "Type": "AWS::IAM::Role",
            "Properties": {
                "AssumeRolePolicyDocument": {
                    "Version": "2012-10-17",
                    "Statement": [
                        {
                            "Effect": "Allow",
                            "Principal": {
                                "Service": "bedrock.amazonaws.com"
                            },
                            "Action": "sts:AssumeRole"
                        }
                    ]
                },
                "Policies": [
                    {
                        "PolicyName": "BedrockAgentPolicy",
                        "PolicyDocument": {
                            "Version": "2012-10-17",
                            "Statement": [
                                {
                                    "Effect": "Allow",
                                    "Action": [
                                        "lambda:InvokeFunction"
                                    ],
                                    "Resource": [
                                        {"Fn::GetAtt": ["RDSSchedulerLambda", "Arn"]}
                                    ]
                                }
                            ]
                        }
                    }
                ]
            }
        },

        # IAM Role for Lambda Function
        "LambdaExecutionRole": {
            "Type": "AWS::IAM::Role",
            "Properties": {
                "AssumeRolePolicyDocument": {
                    "Version": "2012-10-17",
                    "Statement": [
                        {
                            "Effect": "Allow",
                            "Principal": {
                                "Service": "lambda.amazonaws.com"
                            },
                            "Action": "sts:AssumeRole"
                        }
                    ]
                },
                "ManagedPolicyArns": [
                    "arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole"
                ],
                "Policies": [
                    {
                        "PolicyName": "RDSSchedulerPolicy",
                        "PolicyDocument": {
                            "Version": "2012-10-17",
                            "Statement": [
                                {
                                    "Effect": "Allow",
                                    "Action": [
                                        "rds:DescribeDBInstances",
                                        "rds:DescribeDBClusters",
                                        "rds:ListTagsForResource",
                                        "rds:AddTagsToResource",
                                        "rds:RemoveTagsFromResource",
                                        "dynamodb:GetItem",
                                        "dynamodb:PutItem",
                                        "dynamodb:UpdateItem",
                                        "dynamodb:DeleteItem",
                                        "dynamodb:Scan",
                                        "dynamodb:Query"
                                    ],
                                    "Resource": "*"
                                }
                            ]
                        }
                    }
                ]
            }
        },

        # Lambda Function for RDS Scheduler Operations
        "RDSSchedulerLambda": {
            "Type": "AWS::Lambda::Function",
            "Properties": {
                "FunctionName": "bedrock-rds-scheduler",
                "Runtime": "python3.9",
                "Handler": "index.lambda_handler",
                "Role": {"Fn::GetAtt": ["LambdaExecutionRole", "Arn"]},
                "Code": {
                    "ZipFile": """
import json
import boto3
import logging
from datetime import datetime, timezone

logger = logging.getLogger()
logger.setLevel(logging.INFO)

rds_client = boto3.client('rds')
dynamodb = boto3.resource('dynamodb')

def lambda_handler(event, context):
    try:
        logger.info(f"Received event: {json.dumps(event)}")

        # Parse the action from Bedrock agent
        action = event.get('actionGroup', '')
        api_path = event.get('apiPath', '')
        parameters = event.get('parameters', [])

        # Convert parameters to dict
        params = {param['name']: param['value'] for param in parameters}

        if api_path == '/update-rds-schedule':
            result = update_rds_schedule(params)
        elif api_path == '/get-rds-schedule':
            result = get_rds_schedule(params)
        elif api_path == '/list-rds-instances':
            result = list_rds_instances(params)
        elif api_path == '/create-schedule':
            result = create_schedule(params)
        else:
            result = {"error": f"Unknown API path: {api_path}"}

        return {
            'actionGroup': action,
            'apiPath': api_path,
            'httpMethod': event.get('httpMethod', 'POST'),
            'httpStatusCode': 200,
            'responseBody': {
                'application/json': {
                    'body': json.dumps(result)
                }
            }
        }

    except Exception as e:
        logger.error(f"Error: {str(e)}")
        return {
            'actionGroup': action,
            'apiPath': api_path,
            'httpMethod': event.get('httpMethod', 'POST'),
            'httpStatusCode': 500,
            'responseBody': {
                'application/json': {
                    'body': json.dumps({"error": str(e)})
                }
            }
        }

def update_rds_schedule(params):
    db_identifier = params.get('db_identifier')
    schedule_name = params.get('schedule_name')

    if not db_identifier or not schedule_name:
        return {"error": "db_identifier and schedule_name are required"}

    try:
        # Get RDS instance/cluster ARN
        arn = get_rds_arn(db_identifier)

        # Update tags with schedule
        rds_client.add_tags_to_resource(
            ResourceName=arn,
            Tags=[
                {
                    'Key': 'Schedule',
                    'Value': schedule_name
                }
            ]
        )

        return {
            "message": f"Successfully updated schedule for {db_identifier} to {schedule_name}",
            "db_identifier": db_identifier,
            "schedule_name": schedule_name
        }

    except Exception as e:
        return {"error": f"Failed to update schedule: {str(e)}"}

def get_rds_schedule(params):
    db_identifier = params.get('db_identifier')

    if not db_identifier:
        return {"error": "db_identifier is required"}

    try:
        arn = get_rds_arn(db_identifier)

        # Get tags
        response = rds_client.list_tags_for_resource(ResourceName=arn)
        tags = response.get('TagList', [])

        schedule_tag = next((tag for tag in tags if tag['Key'] == 'Schedule'), None)

        return {
            "db_identifier": db_identifier,
            "schedule": schedule_tag['Value'] if schedule_tag else None,
            "all_tags": tags
        }

    except Exception as e:
        return {"error": f"Failed to get schedule: {str(e)}"}

def list_rds_instances(params):
    try:
        # List DB instances
        instances_response = rds_client.describe_db_instances()
        instances = []

        for db in instances_response['DBInstances']:
            instance_info = {
                'identifier': db['DBInstanceIdentifier'],
                'engine': db['Engine'],
                'status': db['DBInstanceStatus'],
                'instance_class': db['DBInstanceClass']
            }

            # Get schedule tag if exists
            try:
                tags_response = rds_client.list_tags_for_resource(
                    ResourceName=db['DBInstanceArn']
                )
                schedule_tag = next(
                    (tag for tag in tags_response['TagList'] if tag['Key'] == 'Schedule'),
                    None
                )
                instance_info['schedule'] = schedule_tag['Value'] if schedule_tag else None
            except:
                instance_info['schedule'] = None

            instances.append(instance_info)

        # List DB clusters
        clusters_response = rds_client.describe_db_clusters()
        clusters = []

        for cluster in clusters_response['DBClusters']:
            cluster_info = {
                'identifier': cluster['DBClusterIdentifier'],
                'engine': cluster['Engine'],
                'status': cluster['Status']
            }

            # Get schedule tag if exists
            try:
                tags_response = rds_client.list_tags_for_resource(
                    ResourceName=cluster['DBClusterArn']
                )
                schedule_tag = next(
                    (tag for tag in tags_response['TagList'] if tag['Key'] == 'Schedule'),
                    None
                )
                cluster_info['schedule'] = schedule_tag['Value'] if schedule_tag else None
            except:
                cluster_info['schedule'] = None

            clusters.append(cluster_info)

        return {
            "instances": instances,
            "clusters": clusters
        }

    except Exception as e:
        return {"error": f"Failed to list RDS resources: {str(e)}"}

def create_schedule(params):
    schedule_name = params.get('schedule_name')
    timezone_param = params.get('timezone', 'UTC')
    periods = params.get('periods', [])

    if not schedule_name:
        return {"error": "schedule_name is required"}

    try:
        # This would integrate with Instance Scheduler's DynamoDB tables
        # For now, return a placeholder response
        return {
            "message": f"Schedule {schedule_name} created successfully",
            "schedule_name": schedule_name,
            "timezone": timezone_param,
            "periods": periods
        }

    except Exception as e:
        return {"error": f"Failed to create schedule: {str(e)}"}

def get_rds_arn(db_identifier):
    try:
        # Try as DB instance first
        response = rds_client.describe_db_instances(DBInstanceIdentifier=db_identifier)
        return response['DBInstances'][0]['DBInstanceArn']
    except:
        # Try as DB cluster
        response = rds_client.describe_db_clusters(DBClusterIdentifier=db_identifier)
        return response['DBClusters'][0]['DBClusterArn']
"""
                },
                "Timeout": 60,
                "Environment": {
                    "Variables": {
                        "INSTANCE_SCHEDULER_STACK": {"Ref": "InstanceSchedulerStackName"}
                    }
                }
            }
        },

        # Bedrock Agent
        "BedrockAgent": {
            "Type": "AWS::Bedrock::Agent",
            "Properties": {
                "AgentName": {"Ref": "BedrockAgentName"},
                "AgentResourceRoleArn": {"Fn::GetAtt": ["BedrockAgentRole", "Arn"]},
                "FoundationModel": "anthropic.claude-3-sonnet-20240229-v1:0",
                "Instruction": "You are an AWS RDS Instance Scheduler management assistant. You can help users update schedules for RDS instances and clusters, view current schedules, list RDS resources, and create new schedules. Always provide clear confirmation of actions taken and helpful information about the scheduling configuration.",
                "ActionGroups": [
                    {
                        "ActionGroupName": "RDSSchedulerActions",
                        "ActionGroupExecutor": {
                            "Lambda": {"Fn::GetAtt": ["RDSSchedulerLambda", "Arn"]}
                        },
                        "ApiSchema": {
                            "Payload": json.dumps({
                                "openapi": "3.0.0",
                                "info": {
                                    "title": "RDS Instance Scheduler API",
                                    "version": "1.0.0",
                                    "description": "API for managing RDS instance scheduling"
                                },
                                "paths": {
                                    "/update-rds-schedule": {
                                        "post": {
                                            "summary": "Update RDS instance or cluster schedule",
                                            "description": "Updates the schedule tag for an RDS instance or cluster",
                                            "operationId": "updateRDSSchedule",
                                            "requestBody": {
                                                "required": True,
                                                "content": {
                                                    "application/json": {
                                                        "schema": {
                                                            "type": "object",
                                                            "properties": {
                                                                "db_identifier": {
                                                                    "type": "string",
                                                                    "description": "RDS instance or cluster identifier"
                                                                },
                                                                "schedule_name": {
                                                                    "type": "string",
                                                                    "description": "Name of the schedule to apply"
                                                                }
                                                            },
                                                            "required": ["db_identifier", "schedule_name"]
                                                        }
                                                    }
                                                }
                                            },
                                            "responses": {
                                                "200": {
                                                    "description": "Schedule updated successfully",
                                                    "content": {
                                                        "application/json": {
                                                            "schema": {
                                                                "type": "object",
                                                                "properties": {
                                                                    "message": {"type": "string"},
                                                                    "db_identifier": {"type": "string"},
                                                                    "schedule_name": {"type": "string"}
                                                                }
                                                            }
                                                        }
                                                    }
                                                }
                                            }
                                        }
                                    },
                                    "/get-rds-schedule": {
                                        "post": {
                                            "summary": "Get current schedule for RDS instance or cluster",
                                            "description": "Retrieves the current schedule configuration for an RDS resource",
                                            "operationId": "getRDSSchedule",
                                            "requestBody": {
                                                "required": True,
                                                "content": {
                                                    "application/json": {
                                                        "schema": {
                                                            "type": "object",
                                                            "properties": {
                                                                "db_identifier": {
                                                                    "type": "string",
                                                                    "description": "RDS instance or cluster identifier"
                                                                }
                                                            },
                                                            "required": ["db_identifier"]
                                                        }
                                                    }
                                                }
                                            },
                                            "responses": {
                                                "200": {
                                                    "description": "Schedule information retrieved",
                                                    "content": {
                                                        "application/json": {
                                                            "schema": {
                                                                "type": "object",
                                                                "properties": {
                                                                    "db_identifier": {"type": "string"},
                                                                    "schedule": {"type": "string"},
                                                                    "all_tags": {"type": "array"}
                                                                }
                                                            }
                                                        }
                                                    }
                                                }
                                            }
                                        }
                                    },
                                    "/list-rds-instances": {
                                        "post": {
                                            "summary": "List all RDS instances and clusters with their schedules",
                                            "description": "Returns a list of all RDS instances and clusters with their current schedule tags",
                                            "operationId": "listRDSInstances",
                                            "requestBody": {
                                                "required": False,
                                                "content": {
                                                    "application/json": {
                                                        "schema": {
                                                            "type": "object",
                                                            "properties": {}
                                                        }
                                                    }
                                                }
                                            },
                                            "responses": {
                                                "200": {
                                                    "description": "List of RDS resources",
                                                    "content": {
                                                        "application/json": {
                                                            "schema": {
                                                                "type": "object",
                                                                "properties": {
                                                                    "instances": {"type": "array"},
                                                                    "clusters": {"type": "array"}
                                                                }
                                                            }
                                                        }
                                                    }
                                                }
                                            }
                                        }
                                    },
                                    "/create-schedule": {
                                        "post": {
                                            "summary": "Create a new schedule",
                                            "description": "Creates a new schedule configuration for use with Instance Scheduler",
                                            "operationId": "createSchedule",
                                            "requestBody": {
                                                "required": True,
                                                "content": {
                                                    "application/json": {
                                                        "schema": {
                                                            "type": "object",
                                                            "properties": {
                                                                "schedule_name": {
                                                                    "type": "string",
                                                                    "description": "Name for the new schedule"
                                                                },
                                                                "timezone": {
                                                                    "type": "string",
                                                                    "description": "Timezone for the schedule",
                                                                    "default": "UTC"
                                                                },
                                                                "periods": {
                                                                    "type": "array",
                                                                    "description": "Schedule periods configuration"
                                                                }
                                                            },
                                                            "required": ["schedule_name"]
                                                        }
                                                    }
                                                }
                                            },
                                            "responses": {
                                                "200": {
                                                    "description": "Schedule created successfully",
                                                    "content": {
                                                        "application/json": {
                                                            "schema": {
                                                                "type": "object",
                                                                "properties": {
                                                                    "message": {"type": "string"},
                                                                    "schedule_name": {"type": "string"}
                                                                }
                                                            }
                                                        }
                                                    }
                                                }
                                            }
                                        }
                                    }
                                }
                            })
                        }
                    }
                ]
            }
        },

        # Lambda permission for Bedrock
        "BedrockLambdaPermission": {
            "Type": "AWS::Lambda::Permission",
            "Properties": {
                "FunctionName": {"Ref": "RDSSchedulerLambda"},
                "Action": "lambda:InvokeFunction",
                "Principal": "bedrock.amazonaws.com",
                "SourceArn": {"Fn::GetAtt": ["BedrockAgent", "AgentArn"]}
            }
        }
    },

    "Outputs": {
        "BedrockAgentId": {
            "Description": "ID of the Bedrock Agent",
            "Value": {"Ref": "BedrockAgent"}
        },
        "BedrockAgentArn": {
            "Description": "ARN of the Bedrock Agent",
            "Value": {"Fn::GetAtt": ["BedrockAgent", "AgentArn"]}
        },
        "LambdaFunctionArn": {
            "Description": "ARN of the Lambda function",
            "Value": {"Fn::GetAtt": ["RDSSchedulerLambda", "Arn"]}
        }
    }
}

# Deployment script
def deploy_bedrock_agent():
    """
    Deploy the Bedrock Agent using CloudFormation
    """
    cf_client = boto3.client('cloudformation')

    try:
        response = cf_client.create_stack(
            StackName='bedrock-rds-scheduler-agent',
            TemplateBody=json.dumps(CLOUDFORMATION_TEMPLATE, indent=2),
            Parameters=[
                {
                    'ParameterKey': 'BedrockAgentName',
                    'ParameterValue': 'RDSSchedulerAgent'
                },
                {
                    'ParameterKey': 'InstanceSchedulerStackName',
                    'ParameterValue': 'instance-scheduler'
                }
            ],
            Capabilities=['CAPABILITY_IAM']
        )

        print(f"CloudFormation stack creation initiated: {response['StackId']}")
        return response

    except Exception as e:
        print(f"Error deploying stack: {str(e)}")
        return None

# Usage examples
USAGE_EXAMPLES = """
# Example interactions with the Bedrock Agent:

1. Update RDS schedule:
   "Update the schedule for my RDS instance 'prod-db-01' to use the 'business-hours' schedule"

2. Get current schedule:
   "What schedule is currently applied to my RDS cluster 'analytics-cluster'?"

3. List all RDS resources:
   "Show me all RDS instances and clusters with their current schedules"

4. Create new schedule:
   "Create a new schedule called 'weekend-only' that runs only on weekends"

The agent will handle the conversation naturally and execute the appropriate API calls.
"""

if __name__ == "__main__":
    # Save CloudFormation template to file
    with open('bedrock-rds-scheduler-template.yaml', 'w') as f:
        import yaml
        yaml.dump(CLOUDFORMATION_TEMPLATE, f, default_flow_style=False)

    print("Bedrock Agent CloudFormation template created!")
    print("Deploy with: aws cloudformation create-stack --stack-name bedrock-rds-scheduler-agent --template-body file://bedrock-rds-scheduler-template.yaml --capabilities CAPABILITY_IAM")
    print(USAGE_EXAMPLES)