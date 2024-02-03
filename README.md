# Erinyes

This repository contains artifacts for the paper:"ERINYES: High Accuracy Attack Provenance of Serverless Intrusions".

## Dependencies

- Go 1.16
- MySQL 8.0.28

## How to use?

This repository comprises two core components of the code: the **Provenance Graph Builder** and the **Execution Partition Engine**. The `express` folder contains the code for **Execution Partition Engine** in the Express.js framework, while the remaining files pertain to **Provenance Graph Builder**.

The Provenance Graph Builder includes the following commands:

- graph: Generate graph and save in database.
- dot: Generate a visualization DOT file based on graph data in the database.
- service: Start a http service to accept logs and build graph.
- subgraph: Trace back and generate a subgraph based on abnormal nodes.

