# Frontend API Client Stub
`frontendclient` is a fake client for the Frontend API. It pretends to be a number of real game clients connecting to Open Match and requests a match, as a group. It then waits for results to come back from the Frontend API, and prints them to your screen.  You can generate these results using the entire Open Match end-to-end workflow - querying the backend, running an MMF, and assigning players to a match - or you can manually test the results pathway by directly putting the results you want into redis.

Only to be used for testing, and only in isolated environments (not in production!)

# Ping files
This application requires files filled with statistics used to generate simulated client latencies.  Example files with placeholder data (with every possible route taking 75ms) are provided.  More realistic data can be found on public sites such as https://wondernetwork.com/wonderproxy or you can provide your own in the format below.

## Format
Ping files are tab-delimited data, in this order:
`City	Distance	Average	% of SOLf/o	min	max	mdev	Last Checked`

## city.percent file
This is a file with of all the cities you want to generate simulated clients from, and associated numbers representing the percent of clients to simulate from those cities. All the percentages are normalized to fall within the range of 0 - 1 (floating point). Developers who want to modify the file should consult the source code to learn more of how it works and is used. 

# Building
Easiest using Google Cloud Build:
```
gcloud builds submit --substitutions TAG_NAME=latest --config cloudbuild.yaml
```

# Further reading
More information about this program can be found in the [development guide](../../docs/development.md).
