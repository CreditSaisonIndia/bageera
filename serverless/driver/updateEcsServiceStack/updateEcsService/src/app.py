import boto3
import json
from collections import defaultdict
from datetime import datetime, timezone, timedelta
import logging
import os


BAGEERA_CLUSTER_ARN = os.getenv('BAGEERA_CLUSTER_ARN')
BAGEERA_PURSUER_SERVICE_ARN = os.getenv('BAGEERA_PURSUER_SERVICE_ARN')
ENV = os.getenv('ENV')

logger = logging.getLogger()
logger.setLevel(logging.DEBUG)

def lambda_handler(event, context):
    
    
    logger.info("Event : %s", json.dumps(event))

    try:
        ecs = boto3.client('ecs')
        cluster_name = BAGEERA_CLUSTER_ARN
        service_name = BAGEERA_PURSUER_SERVICE_ARN
        desired_count = 0

        ecs.update_service(
            cluster=cluster_name,
            service=service_name,
            desiredCount=desired_count
        )

        return {
            'statusCode': 200,
            'body': 'ECS service count updated successfully!'
        }
    except Exception as e:
        logger.error("Error: %s", str(e))
        return {
            'statusCode': 500,
            'body': 'Error updating ECS service count'
        }

    
    
    

    