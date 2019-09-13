FROM microsoft/aspnetcore-build:2.0 AS build-env
WORKDIR /app

COPY . ./
RUN dotnet build -o .
RUN rm -rf obj/ *.pdb *.deps.json
