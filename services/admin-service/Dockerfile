FROM python:3.12-slim
WORKDIR /app
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt
COPY main.py .
COPY static ./static
RUN mkdir -p /data && chown -R 65534:65534 /data
USER 65534:65534
CMD ["uvicorn","main:app","--host","0.0.0.0","--port","8080"]
