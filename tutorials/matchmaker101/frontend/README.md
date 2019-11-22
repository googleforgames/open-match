This folder provides a sample Frontend for Open Match Matchmaker 101 Tutorial.

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

- For GKE users, run:
    ```
    sed "s|REGISTRY_PLACEHOLDER|$REGISTRY|g" frontend.yaml | kubectl apply -f -
    ```
- For Minikube users, [use local image](https://kubernetes.io/docs/setup/learning-environment/minikube/#use-local-images-by-re-using-the-docker-daemon) by running the following command:
    ```bash
    eval $(minikube docker-env)
    sed "s|REGISTRY_PLACEHOLDER|$REGISTRY|g" frontend.yaml | sed "s|Always|Never|g" | kubectl apply -f -
    ```