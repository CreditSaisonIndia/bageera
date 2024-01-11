import json
import logging
import os
import re

import hmacValidation as hv
import http_client
import validation as validate

from UNC_withdrawal_instructions_validator import UNC_WITHDRAWAL_INSTRUCTIONS_SCHEMA
from MTC_withdrawal_instructions_validator import MTC_WITHDRAWAL_INSTRUCTIONS_SCHEMA
from PCC_withdrawal_instructions_validator import PCC_WITHDRAWAL_INSTRUCTIONS_SCHEMA
from withdrawal_instructions_validator import WITHDRAWAL_INSTRUCTIONS_SCHEMA
from CRC_withdrawal_instructions_validator import CRC_WITHDRAWAL_INSTRUCTIONS_SCHEMA

TENURE_MANDATORY = " Validation Failed: tenure is mandatory field"
ROI_MANDATORY = " Validation Failed: loanIntRate is mandatory field"

# API version
API_VERSION = '/api/v1/'

# Setting up logging
logger = logging.getLogger()
logger.setLevel(logging.DEBUG)

MRBURNS_BASE_URL = os.getenv('Mrburns_BASE_URL')
WITHDRAWAL_INSTRUCTION="withdrawal/instructions"
DECIMAL_NUMBER_TEN_INTEGER_TWO_DECIMAL_WITH_DECIMAL_OPTIONAL = "^([0-9]{1,10})(\\.([0-9]{1,2}))?$"

def lambda_handler(event, context):
    logger.debug("Recording with event %s", event)

    hmac_validation_response = hv.validate_request(event)
    if hmac_validation_response["passed"] is False:
        return create_response(400, {'message': hmac_validation_response["error"]})

    if "body" in event and event["body"] is not None:
        request_body = event["body"]
    else:
        return create_response(400, {"message": "Empty Request Body received" })

    request_body_json = json.loads(request_body)

    lpc = validate.fetch_key_value(request_body_json, "lpc")
    schema = None
    if lpc == "MTC":
        schema = MTC_WITHDRAWAL_INSTRUCTIONS_SCHEMA
    elif lpc == "PCC":
        schema = PCC_WITHDRAWAL_INSTRUCTIONS_SCHEMA
    elif lpc == "CRC":
        schema = CRC_WITHDRAWAL_INSTRUCTIONS_SCHEMA
    elif lpc == "UNC":
        schema = UNC_WITHDRAWAL_INSTRUCTIONS_SCHEMA
    else:
        schema = WITHDRAWAL_INSTRUCTIONS_SCHEMA

    validation_response = validate.validate_request_body(event, schema);
    if validation_response["passed"] is False:
        return create_response(400, validation_response["error"])
    
    if "loanComponents" in request_body_json:
        response = validate_loan_components(request_body_json["loanComponents"])
        if "pass" != response:
            return response

    withdrawal_partner_loan_id=fetch_withdrawal_partner_loan_id(request_body)
    murburs_withdrawal_response=fetch_withdrawal_response_from_mrburns(request_body,withdrawal_partner_loan_id)
    return parse_mrburns_response(murburs_withdrawal_response)

def validate_loan_components(loan_components):
    for key in loan_components.keys():
        node = loan_components[key]
        if (("total" not in node) or ("ksfShare" not in node) or ("partnerShare"  not in node))  :
            return create_response(400, {"message": "total, ksfShare, partnerShare are mandatory keys in " + key})
        logger.info(is_number(node["ksfShare"]))
        if is_number(node["total"]) == False or is_number(node["ksfShare"]) == False or is_number(node["partnerShare"]) == False:
            return create_response(400, {"message": "total,ksfShare and partnerShare must be of number type in " + key})
        total = regex_match(DECIMAL_NUMBER_TEN_INTEGER_TWO_DECIMAL_WITH_DECIMAL_OPTIONAL,str(node["total"]))
        ksfShare = regex_match(DECIMAL_NUMBER_TEN_INTEGER_TWO_DECIMAL_WITH_DECIMAL_OPTIONAL,str(node["ksfShare"]))
        partnerShare = regex_match(DECIMAL_NUMBER_TEN_INTEGER_TWO_DECIMAL_WITH_DECIMAL_OPTIONAL,str(node["partnerShare"]))
        if total == False or ksfShare == False or partnerShare == False:
            return create_response(400, {"message": "total,ksfShare and partnerShare can only contain a decimal value with up to 10 digits before and 2 after the decimal point (Negative values not allowed) in " + key})
    return "pass"

def regex_match(regex, string_to_compare):
    if re.search(regex, string_to_compare):
        return True
    else:
        return False

def is_number(val):
    try:
        if type(val) is float or type(val) is int :
            return True
        else :
            return False
    except Exception:
        return False

def fetch_withdrawal_partner_loan_id(request_body):
    request_body = json.loads(request_body)
    if "partnerLoanId" in request_body:
        return request_body["partnerLoanId"]
    return ""

def fetch_withdrawal_response_from_mrburns(request_body,withdrawal_partner_loan_id):
    logger.info("Hitting MrBurns for partner_loan_id %s to fetch withdrawal instruction result ", withdrawal_partner_loan_id)
    url = MRBURNS_BASE_URL + API_VERSION + WITHDRAWAL_INSTRUCTION
    logger.info("Hitting Mrburns api-  %s", url)
    return http_client.post(url=url, data=request_body)

def parse_mrburns_response(murburs_withdrawal_response):
    if murburs_withdrawal_response.status_code >= 500 and murburs_withdrawal_response.status_code <= 599 :
        logger.error("Could not fetch MrBurns Response due to- %s", murburs_withdrawal_response.content)
        return create_response(503,{"result":"Service Unavailable"})
    murburs_withdrawal_body=murburs_withdrawal_response.json()
    logger.debug("withdrawal instruction response %s",murburs_withdrawal_body)
    if(murburs_withdrawal_body["message"] == TENURE_MANDATORY or murburs_withdrawal_body["message"] == ROI_MANDATORY) :
        return create_response(400, {'message': "Structure is not proper"})
    return create_response(murburs_withdrawal_response.status_code,murburs_withdrawal_body)

def create_response(status_code, response):
    return {
        "statusCode": status_code,
        "body": json.dumps(response),
    }
