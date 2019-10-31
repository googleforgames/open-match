This folder provides a sample Frontend for Open Match Matchmaker 101 Tutorial.
<TODO - Update the README with details of the Frontend and the steps that need
to be run before executing the commamds below>

Run the below steps in this folder to set up the Frontend.

Step1: Specify your Registry URL.
```
REGISTRY=[YOUR_REGISTRY_URL]
```

Step2: Build the Frontend image.
```
docker build -t $REGISTRY/mm101-tutorial-frontend .
```

Step3: Push the Frontend image to the configured Registry.
```
docker push $REGISTRY/mm101-tutorial-frontend
```

Step4: Update the install yaml for your setup.
```
sed "s|REGISTRY_PLACEHOLDER|$REGISTRY|g" frontend.yaml | kubectl apply -f -
```
