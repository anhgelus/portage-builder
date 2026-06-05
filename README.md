# Portage Builder

Build Gentoo packages remotely with your Portage's configuration.

## How does it work?

The client connects to the builder (called the server).
They can update their configuration or request the build of specific packages.
When the server has finished building, the packages are available in a Portage binary repo.

It relies on a custom protocol to be as fast as possible.
It uses DAG-CBOR to encode and decode arguments to reduce the network overhead.

The official implementation is using SSH to send information between the server and the client.
Clients can authentificate with public keys.
You can reimplement the protocol with HTTP, Gemini, Gopher or any other layer-7 application.

## Usage

Portage Builder is split between a client (folder `client`) and a server (folder `server`).
The server is building the packages for the client.

A CLI is available to send requests to the server.
A daemon is present to run your own server.

The configuration is written in TOML.

See the man pages `portage-builder(1)` and `portage-builderd(1)` for more information.

## Protocol

The formal grammar is in [grammar.abnf](./grammar.abnf).
The official implementation is in `proto`.
