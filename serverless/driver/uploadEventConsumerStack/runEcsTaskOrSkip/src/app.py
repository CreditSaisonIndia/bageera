import boto3
import json
from collections import defaultdict
from datetime import datetime, timezone, timedelta
import logging
import os

PQ_JOB_QUEUE_URL = os.getenv('PQ_JOB_QUEUE_URL')
BAGEERA_CLUSTER_ARN = os.getenv('BAGEERA_CLUSTER_ARN')
BAGEERA_ECS_JOB_SG_ID = os.getenv('BAGEERA_ECS_JOB_SG_ID')
BAGEERA_JOB_DEFINITION_ARN = "bageera-job-definition"
ENV = os.getenv('ENV')
SERVICE_SUBNETS = os.getenv('SERVICE_SUBNETS')
ALERT_SNS_ARN = os.getenv('ALERT_SNS')

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
    task_definition = BAGEERA_JOB_DEFINITION_ARN
    cluster = BAGEERA_CLUSTER_ARN
    launch_type = 'FARGATE'  # or EC2 depending on your setup
    subnet = SERVICE_SUBNETS.split(",")
    security_group = BAGEERA_ECS_JOB_SG_ID

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
                    'name': 'alertSnsArn',
                    'value': ALERT_SNS_ARN,
                },
                {
                    'name': 'environment',
                    'value': ENV,
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
    task_definition_arn=BAGEERA_JOB_DEFINITION_ARN

    # Check if any tasks match the specified task definition ARN
    # matching_tasks = [task for task in tasks.get('taskArns', []) if ecs.describe_tasks(cluster=cluster, tasks=[task])['tasks'][0]['taskDefinitionArn'].startswith(task_definition_arn)]
    matching_tasks=[]
    for task in tasks.get('taskArns', []):
        task_description = ecs.describe_tasks(cluster=cluster, tasks=[task])['tasks'][0]
        if task_definition_arn in task_description['taskDefinitionArn']:
            matching_tasks.append(task)
        
    print(matching_tasks)

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
                    'subnets': subnet,
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
            
            logger.info("Task started successfully")
            
            logger.info("RUN TASK RESPONSE : ", response)
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

    
    
    

    