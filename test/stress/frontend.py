from locust import HttpLocust, TaskSequence, task, seq_task
import random
import string
import json

ID_LEN = 6

class ClientBehavior(TaskSequence):

  def on_start(self):
    """ on_start is called when a Locust start before any task is scheduled """
    self.init()

  def init(self):
    # Placeholder for initialize future TLS materials and request generators
    random_generator = lambda len : ''.join(random.choices(string.ascii_uppercase + string.digits, k=len))
    self.player_generator = lambda : {
        "player": {
            "id": random_generator(ID_LEN),
            "properties": json.dumps({
                "level": random_generator(ID_LEN),
                "strength": random_generator(ID_LEN),
                "agility": random_generator(ID_LEN)
            })
        }
    }

    self.payload = {
        "endpoint": "/v1/frontend/players",
        "params": None,
        "body": None
    }

  @seq_task(1)
  @task(5)
  def create_player(self):
    self.payload["body"] = self.player_generator()

    method, endpoint, params, data, name = "PUT", self.payload["endpoint"], None, json.dumps(self.payload["body"]), "Create: {}".format(self.payload["endpoint"])

    with self.client.request(method, endpoint, name=name, params=params, data=data, catch_response=True) as response:
      if response.content != b"{}":
        response.failure("Got {}, was expected {{}}".format(response.content))

  @seq_task(2)
  @task(5)
  def delete_player(self):
    method, endpoint, params, data, name = "DELETE", "{}/{}".format(self.payload["endpoint"], self.payload["body"]["player"]["id"]), None, None, "Delete: {}/[id]".format(self.payload["endpoint"])

    with self.client.request(method, endpoint, name=name, params=params, data=data, catch_response=True) as response:
      if response.content != b"{}":
        response.failure("Got {}, was expected {{}}".format(response.content))



class WebsiteUser(HttpLocust):
  task_set = ClientBehavior
  min_wait = 500
  max_wait = 1500
