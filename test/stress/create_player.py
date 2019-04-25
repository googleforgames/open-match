from locust import HttpLocust, TaskSet, task

class ClientBehavior(TaskSet):
  def on_start(self):
    """ on_start is called when a Locust start before any task is scheduled """
    self.init()

  def init(self):
    # Placeholder for initialize future TLS materials and request generators

  @task(1)
  def create_player(self):
    # Placeholder for calling CreatePlayer function in frontendapi

  @task(1)
  def delete_player(self):
    # Placeholder for calling DeletePlayer function in frontendapi


class WebsiteUser(HttpLocust):
  task_set = ClientBehavior
  min_wait = 1000
  max_wait = 3000
