# Copyright 2019 Google LLC
# 
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
# 
#     http://www.apache.org/licenses/LICENSE-2.0
# 
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
import random
import json

from locust import HttpLocust, TaskSequence, task, seq_task
from util import ticket_generator, pool_generator, ATTRIBUTE_LIST


NUM_QUERY_ATTR = 20

class ClientBehavior(TaskSequence):

  def on_start(self):
    """ on_start is called when a Locust start before any task is scheduled """
    self.init()

  def init(self):
    # Placeholder for initialize future TLS materials and request generators
    create_payload = {
      "method": "POST",
      "endpoint": "/v1/frontend/tickets",
      "params": None,
      "body": None
    }

    # Each spawned client first generate 10 tickets then do the query to mmlogic (data) layer
    # Total number of tickets in open-match would be 10 * # of spawned clients
    for i in range(10):
        self.client.request(create_payload["method"], create_payload["endpoint"], params=None, data=json.dumps(ticket_generator()))


  @task(1)
  def query_ticket(self):
    query_payload = {
      "method": "POST",
      "endpoint": "/v1/mmlogic/tickets:query",
      "params": None,
      "body": pool_generator(random.choices(ATTRIBUTE_LIST, k=NUM_QUERY_ATTR))
    }
    method, endpoint, params, data, name = query_payload["method"], query_payload["endpoint"], None, json.dumps(query_payload["body"]), "Query: {}".format(query_payload["endpoint"])

    with self.client.request(method, endpoint, name=name, params=params, data=data, catch_response=True) as response:
      if response.status_code != 200:
        response.failure("Got status code {}, was expected 200.".format(response.content))


class WebsiteUser(HttpLocust):
  task_set = ClientBehavior
  min_wait = 500
  max_wait = 1500
