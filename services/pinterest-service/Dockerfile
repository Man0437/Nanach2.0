FROM python:3.12-slim

WORKDIR /app

COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

COPY main.py images.json ./

USER 10001

CMD ["uvicorn", "main:app", "--host", "0.0.0.0", "--port", "8000"]