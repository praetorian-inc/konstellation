FROM python:3.11

# Set the working directory to /app
WORKDIR /app

COPY * /app

RUN pip install -r /app/requirements.txt

ENTRYPOINT ["python3", "konstellation.py"]

