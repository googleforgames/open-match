// Copyright 2019 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

window.onload = function() {
  let protocol = "ws://";
  if (window.location.protocol == "https:") {
    protocol = "wss://";
  }
  const ws = new WebSocket(protocol + window.location.host + "/connect");

  ws.onopen = function (event) {
    return false;
  }

  ws.onmessage = function (event) {    
    document.getElementById("content").textContent = event.data;
    return false;
  }

  ws.onerror = function (event) {
    console.log("ERROR!");
    console.log(event);
    document.getElementById("error").textContent = event.toString();
    return false;
  }

  ws.onclose = function (event) {
    console.log("WS CLOSED!");
    console.log(event);
    document.getElementById("error").textContent = event.toString();
    return false;
  }
}
