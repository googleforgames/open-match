-"I notice that all the APIs use gRPC. What if I want to make my calls using REST, or via a Websocket?"
 - (gateway/proxy OSS projects are available)
-"How do I put my players into a queue?"
 - Open Match doesn't have a concept of _queues_ that game clients need to join; instead, your clients should tag themselves with a series of attributes that you then configure your game's online backend to look for.  So, for example, if you want to match players into a Capture the Flag game type, a common approach would look like this:
   - Code your game clients so that players who select the Capture the Flag game type will pass in an associated attribute to the `Frontend.CreatePlayer()` call. You can name the tag any string that can be a Redis key (it is recommended you stick with alphanumerics), but the value must be an integer.  A typical approach uses a positive integer to indicate a selection for flags like mode selection.  You could add a attribute to the input JSON like this:
    ```
    { ...
     "mode" : {
       "ctf": 1
       }
     ...
    }
    ```
     (This example elides all the other properties a player might pass in that your matchmaking function might take into account when trying to find compatible players, but you should include those as well.)
   - Code your game's online backend to ask Open Match to find compatible players  that all share the tag you choose to indicate they want to play Capture the Flag. This can be passed in using the profile JSON when calling `backend.CreateMatch()`.  A typical approach here would include a `PlayerPool` that specifies filters looking for your attribute:
    ```
    { ...
      "properties":{
        "pools": [
           {
             "name": "ctfPool",
             "filters": [
               { "name": "ctf", "attribute": "mode.ctf", "minv": "1", "maxv": "1" },
               ...
             ]
           },
        ...
      }
    ...
    }
    (Again, this example profile elides all the other filters you select for evaluating other player properties such as latency or skill rating, but these filters should also be included.)
   - Write a matchmaking function (MMF) that uses the MMLogic API to get all the players that match your `ctfpool` filters, and then attempts to match together your players based on all the other properties you want to evaluate for compatibility.  If your MMF don't find enough players to fill your match, simply return an error and a message like "insufficient players".
   - Your backend can immediately begin asking Open Match to find matches using this profile by calling `backend.CreateMatch()`, since your MMF will just return the "insufficient players" error if there aren't enough players - your backend can then retry.
   - If Open Match is returning successful sessions, your online backend is responsible for distributing those matches to dedicated game servers (DGS), then calling `backend.CreateAssignments()` to return the DGS connection details to your game clients.
