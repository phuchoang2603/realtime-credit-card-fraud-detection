FROM jupyter/minimal-notebook:python-3.8.8

USER root

# Install system packages
RUN apt-get update && \
  apt-get install -y --no-install-recommends \
  curl \
  graphviz && \
  apt-get clean && \
  rm -rf /var/lib/apt/lists/*

USER $NB_UID

# Copy requirements.txt into image
COPY requirements.txt /tmp/requirements.txt

# Install Python packages
RUN pip install --no-cache-dir -r /tmp/requirements.txt
