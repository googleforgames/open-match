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

FROM python:3.5.3
WORKDIR /usr/src/open-match
COPY examples/functions/python3/mmlogic-simple examples/functions/python3/mmlogic-simple
COPY config config
WORKDIR /usr/src/open-match/examples/functions/python3/mmlogic-simple
RUN pip install --no-cache-dir -r requirements.txt

CMD ["python", "./harness.py"]
