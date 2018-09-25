# Frontend API Client Stub
`frontendclient` is a fake client for the Frontend API. It pretends to be a real game client connecting to Open Match and requests a game, then dumps out the connection string it receives. Note that it doesn't actually test the return path by looking for arbitrary results from your matchmaking function; it pauses and tells you the name of a key to set a connection string in directly using a redis-cli client.

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
More information about this program can be found in the development guide.
