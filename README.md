# RandomSecrets Operator

:warning: **This operator is not intended for production use.**

This operator is a project built for educational purposes. It is a simple operator that modifies a secret to add a random value to it. It is built to demonstrate how to build an operator without using any frameworks.

**Features:**
- Adds a random value to a secret.
- Takes configuration from annotations (if the secret has to be updated, which key to update, if the random value should contain special characters).
- Use a CRD to define the configuration of a secret (which will be created by the operator).
- Use a MutatingWebhook to intercept the creation of the secret and add the random value to it.

## Installation

Install the operator by deploying the Helm chart:

```bash
helm install randomsecrets-operator ./helm
```

## Usage

### With a CRD

Create a `RandomSecret` resource to define the configuration of the secret:

```yaml
apiVersion: secret.a-cup-of.coffee/v1
kind: RandomSecret
metadata:
  name: random-secret
  namespace: default
spec:
  key: password
  secretName: secret-name
  length: 32
  static: 
    username: admin
    email: "quentin@a-cup-of.coffee"
```

### With annotations

Create a secret with the following annotations:

```yaml
apiVersion: v1
kind: Secret
metadata:
  annotations:
    secret.a-cup-of.coffee/enable: "true"
    secret.a-cup-of.coffee/key: password
  name: non-empty-secret
  namespace: coder
type: Opaque
```

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
