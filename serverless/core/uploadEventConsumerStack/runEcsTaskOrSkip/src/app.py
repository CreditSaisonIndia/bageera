import boto3
import json
from collections import defaultdict
from datetime import datetime, timezone, timedelta
import logging
import os

PQ_JOB_QUEUE_URL = os.getenv('PQ_JOB_QUEUE_URL')

logger = logging.getLogger()
logger.setLevel(logging.DEBUG)

def lambda_handler(event, context):
    
    
    logger.info("Message from SNS: %s", json.dumps(event))

    # Extract bucket and object key
    message = json.loads(event["Records"][0]["Sns"]["Message"])
    bucket = message["bucketName"]
    object_key = str(message["objectKey"])

    logging.info("Bucket in SNS message : %s", bucket)
    logging.info("Object Key in SNS Message: %s", object_key)
    
    file_name = os.path.basename(object_key)
    
    # Values to pass to the ECS task
    task_definition = 'bageera-job-definition'
    cluster = 'bageeraEcsCluster'
    launch_type = 'FARGATE'  # or EC2 depending on your setup
    subnet = 'subnet-02bf49ba39bfed0a2'
    security_group = 'sg-0c4dbe2de90e1a522'

    # Additional parameters for your task
    container_overrides = [
        {
            
            'name': 'bageera-job-container',
            'environment': [
                {
                    'name': 'fileName',
                    'value': file_name,
                },
                {
                    'name': 'bucketName',
                    'value': bucket,
                },
                {
                    'name': 'objectKey',
                    'value': object_key,
                },
                {
                    'name': 'requestQueueUrl',
                    'value': PQ_JOB_QUEUE_URL,
                },

                {
                    'name': 'environment',
                    'value': "dev",
                },
            ],
            # No need to specify 'command' if it's already defined in the Dockerfile CMD
        },
        # Add more container overrides if needed
    ]
    # Create ECS client
    ecs = boto3.client('ecs')

    # List all tasks in the cluster
    tasks = ecs.list_tasks(
        cluster=cluster,
        desiredStatus='RUNNING'  # You can adjust the status based on your requirements
    )
    task_definition_arn="arn:aws:ecs:us-east-1:971709774307:task-definition/bageera-job-definition:3"

    # Check if any tasks match the specified task definition ARN
    matching_tasks = [task for task in tasks.get('taskArns', []) if ecs.describe_tasks(cluster=cluster, tasks=[task])['tasks'][0]['taskDefinitionArn'] == task_definition_arn]

    if matching_tasks:
        print(f'Task definition {task_definition_arn} is already running. Skipping launch.')
        return {
            'statusCode': 200,
            'body': f'Task definition {task_definition_arn} is already running. Skipping launch.'
        }
    else:
        run_task_params = {
            'taskDefinition': task_definition,
            'launchType': launch_type,
            'cluster': cluster,
            'networkConfiguration': {
                'awsvpcConfiguration': {
                    'subnets': [subnet],
                    'securityGroups': [security_group],
                },
            },
            'overrides': {
                'containerOverrides': container_overrides,
            },
        }
    
        try:
            # Start the ECS task
            response = ecs.run_task(**run_task_params)
            print(response)
            return {
                'statusCode': 200,
                'body': json.dumps('Task started successfully')
            }
        except Exception as e:
            print(f'Error: {e}')
            return {
                'statusCode': 500,
                'body': json.dumps('Error starting ECS task')
            }

    
    
    

    