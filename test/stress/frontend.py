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
import json

from locust import HttpLocust, TaskSequence, task, seq_task
from util import string_generator, number_generator, ticket_generator

class ClientBehavior(TaskSequence):

  def on_start(self):
    """ on_start is called when a Locust start before any task is scheduled """
    self.init()

  def init(self):
    # Placeholder for initialize future TLS materials and request generators
    self.payload = {
      "endpoint": "/v1/frontend/tickets",
      "params": None,
      "body": None
    }

  @seq_task(1)
  @task(5)
  def create_ticket(self):
    self.payload["body"] = ticket_generator()

    method, endpoint, params, data, name = "POST", self.payload["endpoint"], None, json.dumps(self.payload["body"]), "Create: {}".format(self.payload["endpoint"])
    with self.client.request(method, endpoint, name=name, params=params, data=data, catch_response=True) as response:
      if response.status_code == 200:
          response.success()
          self.payload["body"] = json.loads(response.content)

  @seq_task(2)
  @task(5)
  def delete_ticket(self):
    method, endpoint, params, data, name = "DELETE", "{}/{}".format(self.payload["endpoint"], self.payload["body"]["ticket"]["id"]), None, None, "Delete: {}/[id]".format(self.payload["endpoint"])

    with self.client.request(method, endpoint, name=name, params=params, data=data, catch_response=True) as response:
      if response.content != b"{}":
        response.failure("Got {}, was expected {{}}".format(response.content))


class WebsiteUser(HttpLocust):
  task_set = ClientBehavior
  min_wait = 500
  max_wait = 1500
