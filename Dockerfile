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
    curl \
    wget \
    unzip \
    rsync \
    git \
    ca-certificates \
    openssh-client \
    gpg

# install tfsec
COPY ./signing.asc /signing.asc
RUN gpg --import < signing.asc \
  && curl -fsSL github.com/aquasecurity/tfsec/releases/latest/download/tfsec-linux-amd64 -O \
  && curl -fsSL github.com/aquasecurity/tfsec/releases/latest/download/tfsec-linux-amd64.D66B222A3EA4C25D5D1A097FC34ACEFB46EC39CE.sig -O \
  && gpg --verify tfsec-linux-amd64.D66B222A3EA4C25D5D1A097FC34ACEFB46EC39CE.sig tfsec-linux-amd64 \
  && chmod 755 tfsec-linux-amd64 \
  && mv tfsec-linux-amd64 /usr/local/bin/tfsec \
  && rm tfsec-linux-amd64.D66B222A3EA4C25D5D1A097FC34ACEFB46EC39CE.sig

# install aws-cli v2
RUN curl https://awscli.amazonaws.com/awscli-exe-linux-x86_64.zip -o awscliv2.zip \
  && unzip awscliv2.zip \
  && ./aws/install \
  && rm -rf aws awscliv2.zip

# install ansible and python dependencies
RUN pip install ansible-base==2.10.3 ansible==2.10.3 boto3==1.16.12 botocore==1.19.63 hvac==0.10.5 requests

# install community.general with -no-color removed for terraform
# https://github.com/ansible-collections/community.general/issues/5613
COPY ./community-general-3.2.0.tar.gz /community-general-3.2.0.tar.gz
RUN ansible-galaxy collection install /community-general-3.2.0.tar.gz community.hashi_vault:==2.4.0 community.aws:==4.1.1 amazon.aws:==4.2.0

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
