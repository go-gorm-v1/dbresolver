FROM golang:1.18

WORKDIR /usr/local/app
RUN apt-get update && apt-get install build-essential sqlite3 -y

COPY . .

ENTRYPOINT ["/usr/local/app/test.sh"]
CMD ["all"]
