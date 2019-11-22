This folder provides a sample Evaluator for Open Match Matchmaker 101 Tutorial.

Run the below steps in this folder to set up the Evaluator.

Step1: Specify your Registry URL.
```
REGISTRY=[YOUR_REGISTRY_URL]
```

Step2: Build the Evaluator image.
```
docker build -t $REGISTRY/custom-eval-tutorial-evaluator .
```

Step3: Push the Evaluator image to the configured Registry.
```
docker push $REGISTRY/custom-eval-tutorial-evaluator
```

Step4: Update the install yaml for your setup.

- For GKE users, run:
    ```bash
    sed "s|REGISTRY_PLACEHOLDER|$REGISTRY|g" evaluator.yaml | kubectl apply -f -
    ```
- For Minikube users, [use local image](https://kubernetes.io/docs/setup/learning-environment/minikube/#use-local-images-by-re-using-the-docker-daemon) by running the following command:
    ```bash
    eval $(minikube docker-env)
    sed "s|REGISTRY_PLACEHOLDER|$REGISTRY|g" evaluator.yaml | sed "s|Always|Never|g" | kubectl apply -f -
    ```