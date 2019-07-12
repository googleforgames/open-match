import random
import string
import json

ID_LEN = 6
ATTRIBUTE_MIN = 1
ATTRIBUTE_MAX = 101

def string_generator():
    return ''.join(random.choice(string.ascii_uppercase + string.digits) for x in range(ID_LEN))

def number_generator():
    return random.randint(ATTRIBUTE_MIN, ATTRIBUTE_MAX)

def ticket_generator(): 
    return {
        "ticket": {
            "properties": { attribute: number_generator() for attribute in ATTRIBUTE_LIST}
        }
    }

def pool_generator(attribute_names): 
    return {
        "pool": {
            "name": string_generator(),
            "filter": json.dumps([
                {
                    attribute: number_generator(),
                    "min": number_generator(),
                    "max": number_generator()
                } for attribute in attribute_names
            ])
        }
    }

# Generate 100 attributes for load testing
ATTRIBUTE_LIST = [str(i) + string_generator() for i in range(100)]