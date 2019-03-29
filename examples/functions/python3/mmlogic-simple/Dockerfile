FROM python:3.5.3
WORKDIR /usr/src/open-match
COPY examples/functions/python3/mmlogic-simple examples/functions/python3/mmlogic-simple
COPY config config
WORKDIR /usr/src/open-match/examples/functions/python3/mmlogic-simple
RUN pip install --no-cache-dir -r requirements.txt

CMD ["python", "./harness.py"]
