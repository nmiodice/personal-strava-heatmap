FROM mcr.microsoft.com/azure-functions/python:3.0-python3.8

ENV AzureWebJobsScriptRoot=/home/site/wwwroot \
    AzureFunctionsJobHost__Logging__Console__IsEnabled=true

# Create app directory
WORKDIR /home/site/wwwroot

RUN apt-get update
RUN apt install -y libpq-dev python3-dev
RUN python -m pip install --upgrade pip


# Install dev dependencies
COPY requirements.dev.txt ./
RUN pip install -r requirements.dev.txt

# Install app dependencies
COPY requirements.txt ./
RUN pip install -r requirements.txt

# Bundle app source
COPY . /home/site/wwwroot

RUN (cd queue-trigger && mypy --ignore-missing-imports *.py)
RUN (cd queue-trigger && isort --check-only *.py)
RUN (cd queue-trigger && autopep8 --aggressive --exit-code --in-place *.py)
