# personal-strava-heatmap


## Configure environment

Many of the steps here assume that your shell environment is configured with the appropriate environment variables. You can find the required variables inside the relevant directory. Environment variables can be set using a tool like [direnv](https://direnv.net/), or by running the following:

```bash
DOT_ENV=.env
export $(cat $DOT_ENV | grep -v '^\s*#' | xargs)
```

## Initial Deployment

### Setup Terraform Backend State

Terraform state should live in a remote container in order to be leveraged across a variety of machines (developer workstations, CI agents, etc...). The `terraform-bootstrap` module provisions and configures the state container:

> **Note**: If you're authenticating using a Service Principal then it must have permissions to both `User.Read` and `Application.ReadWrite.OwnedBy` within the `Windows Azure Active Directory` API. This allows it to create AAD applications. It will also need the `owner` in the subscription being deployed to in order to do role assignments.

```bash
cd terraform-bootstrap/
terraform apply -auto-approve

# capture backend state configuration
ARM_ACCESS_KEY=$(terraform output backend-state-account-key)
ARM_ACCOUNT_NAME=$(terraform output backend-state-account-name)
ARM_CONTAINER_NAME=$(terraform output backend-state-container-name)

# capture container registry ID
ACR_ID=$(terraform output acr-id)

cd ..
```

### Deploy Infrastructure

Now that the remote state container is configured it is possible to deploy the infrastructure:

> **Note**: instructions for setting `ARM_ACCESS_KEY`, `ARM_ACCOUNT_NAME`, `ARM_CONTAINER_NAME` are described above.

```bash
cd terraform
terraform plan
terraform apply
cd ..
```

### Build & Deploy Image Processor (Azure Function)

The image processing component is responsible for converting ingested ride data into map tiles. This is modeled as an Azure Function because it needs to handle large compute batches that happen in large bursts. Azure Functions scale up to meet this demaind, and scale down when they are not needed.

```bash
./scripts/build_function.sh
./scripts/deploy_function.sh
```

### Build & Deploy API server

The API server handles most user interaction and can also kick off map computation workloads.

```bash
./scripts/build_service.sh

# Deploy is TBD
```
