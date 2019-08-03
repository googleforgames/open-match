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
import os
import glob
from datetime import datetime

from pathlib import Path
from google.cloud import storage
from locust import HttpLocust, TaskSequence, task, seq_task, events
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
          self.payload["body"] = json.loads(str(response.content, 'utf-8'))

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


"""
*on_quitting* is fired when the locust process is exiting
"""
def on_quitting(**kw):
    """ Upload data to GCS bucket on quit """
    storage_client = storage.Client()

    bucket = storage_client.get_bucket("artifacts.{}.appspot.com".format(os.environ['GCP_PROJECT']))

    rdir = os.path.dirname(os.path.realpath(__file__))
    rfiles = glob.glob(rdir + '/*.csv')

    for rfile in rfiles:
      time_postfix = str(datetime.now().replace(microsecond=0))
      blobname = Path(rfile).stem + '_' + time_postfix + '.csv'
      blob = bucket.blob('stress/' + blobname)
      blob.upload_from_filename(rfile)
      print("Test result is available via {}".format(blob.public_url))

    return

"""
Register locust.events.quitting hook if executing under --no-web mode in a kubernetes container
"""
if "NO_WEB" in os.environ:
  events.quitting += on_quitting
