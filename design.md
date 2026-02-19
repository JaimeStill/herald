# Herald Design

~1,000,000 classified PDF documents that need their classifications interpreted so that database records can be generated for association.

## Project Details

- Go web service with an embedded Lit web components web client built and managed with Bun
- Containerized and hosted in Azure (both IL4 and IL6 Azure) using Azure AI Foundry deployed GPT models. I know gpt-5-mini is available, gpt-5.2 should be available as well. These should be our supported targets, and the web service should).
- Leverage `~/code/go-agents`, `~/code/go-agents-orchestration`, and `~/code/document-context`
- Uses `~/code/agent-lab` as the starting point for the architecture, but constrained around the requirements established for this project.

## Requirements

I know gpt-5-mini is available on IL4 and IL6 Azure AI Foundry; gpt-5.2 should be as well. We'll target both of these models initially, and architect the web service so that it contains just a single configurable agent configuration. That way, we can use external configuration to define the agent based on the deployment environment (similar to a DB connection string).

The documents that need to be classified are not located in any single location, so I want to setup the web service to accommodate two different modes of processing:

1. An endpoint that allows processing a single document
2. The ability to batch process documents uploaded to an Azure Blob Storage container

The Azure Blob Storage container should be uniquely owned and associated with the herald web service, and we should have a conventional approach for managing the documents within the container. This means that we should be able to infer whether a document has been processed or not simply by where it is stored in the Blob Storage container. In addition to the automated movement of documents within the Blob Storage container, we should also be able to manage and adjust these documents through the herald web service API. What would be the optimal approach for consolidating ~1,000,000 documents into blob storage in a way that allows associated metadata to be retained to ensure that the document can be clearly referenced from the API? Should there be a document registration endpoint that allows a registration record to be recorded in the SQL database that provides the unique document properties and allows the document record to have an association with a classification record containing the inferred result (once processed)?

An Azure SQL database should store the inferred classification results as each document is processed, and we should have API endpoints that facilitate querying these results. Users should also be able to validate the classification by viewing the result alongside the PDF document the result was derived from, and make any manual adjustments. See comment in Azure Blob Storage details in the paragraph above regarding document registration with metadata.

Azure Entra auth should be used to protect the Go web service and the Lit web client should support OBO auth to the web service. The web service should have permissions to the Azure SQL database, Azure Blob Storage container, and Azure AI Foundry deployment (although the Azure provider will require either an API key or an Access Token to be provided with the agent configuration; need to identify the best way of handling this. Probably API key stored in Azure Key Vault or a secret in wherever we deploy the container to).

The prompts for each stage of the workflow should have a hard-coded default value, but have the ability to be modified at runtime. We will not use the full on Profile feature from agent-lab, but simple named modifications that can be loaded in for each workflow phase. The workflow itself, derived from the agent-lab implementation, should be optimized and streamlined to avoid heavy processing and token consumption. We should aim for the same results, but with a lighter workload and trimmed down state graph. It will be imperative that we take our time working through the design of this during the project planning phase.

This project needs to be able to be developed very fast, avoiding going down rabbit holes or getting caught up in adding unnecessary features. We have covered the majority of the ground needed during the R&D phases to handle most of what we will need to deliver this. We should avoid the image caching features from the agent-lab project and purely focus on retaining the documents themselves. Once inference has been completed, any generated images should be discarded.
