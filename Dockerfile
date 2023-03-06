FROM public.ecr.aws/lts/ubuntu:bionic

ARG TERRAFORM_VERSION="1.3.9"

# add repositories and update
RUN apt-get update -y && \
    apt-get install -y software-properties-common && \
    apt-add-repository -y ppa:ansible/ansible

# install dependent os packages
RUN apt-get update -y && \
    DEBIAN_FRONTEND="noninteractive" \
    apt-get install -y \
    python \
    python-pip \
    awscli \
    wget \
    unzip \
    rsync \
    git \
    ca-certificates \
    openssh-client

# install ansible and python dependencies
RUN pip install ansible-base==2.10.3 ansible==2.10.3 boto3==1.16.12 botocore==1.19.12 hvac==0.10.5 && ansible-galaxy collection install community.general:==3.2.0 community.hashi_vault:==2.4.0

# write ansible config file
RUN mkdir -p /etc/ansible && \
    echo -e "[local]\nlocalhost ansible_connection=local" > /etc/ansible/hosts

# configure SSH client for private terraform modules
RUN mkdir -p $HOME/.ssh
RUN echo "StrictHostKeyChecking no" >> $HOME/.ssh/config
RUN echo "LogLevel quiet" >> $HOME/.ssh/config
RUN chmod 0600 $HOME/.ssh/config

# download third-pary dependencies
RUN wget -O terraform.zip https://releases.hashicorp.com/terraform/${TERRAFORM_VERSION}/terraform_${TERRAFORM_VERSION}_linux_amd64.zip && \
    unzip terraform.zip && mv terraform /usr/local/bin/terraform && chmod a+x /usr/local/bin/terraform && rm terraform.zip

RUN mkdir -p /opt/resource
COPY ./dist/check_linux_amd64_v1/check /opt/resource/check
COPY ./dist/in_linux_amd64_v1/in /opt/resource/in
COPY ./dist/out_linux_amd64_v1/out /opt/resource/out
COPY ./ansible /opt/ansible

ENTRYPOINT ["/bin/bash"]
