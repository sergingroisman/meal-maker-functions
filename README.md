# Meal Maker API

## Go on Azure Functions and GitHub Actions Setup ✍️

## Prerequisites 🔍

- [Azure CLI](https://docs.microsoft.com/en-us/cli/azure/install-azure-cli)
- [Go SDK](https://go.dev/dl/)
- [Azure Functions Core Tools](https://docs.microsoft.com/en-us/azure/azure-functions/functions-run-local?tabs=v4%2Cmacos%2Ccsharp%2Cportal%2Cbash#install-the-azure-functions-core-tools)
- Prefererrably, Azure Functions [VS Code extension](https://marketplace.visualstudio.com/items?itemName=ms-azuretools.vscode-azurefunctions)

### Deploying 🚀

1. Clone the repo
2. Create a Function App in Azure
3. Configure GitHub Actions secrets
4. Push a commit to the `main` branch

## Contributing ❤️
 - [Sérgio Junior
 ](https://github.com/sergingroisman)


OBS: Caso seja necessário criar um usuário na base do mongodb:
```bash
mongosh -u root -p secret

use meal-maker-db

db.createUser(
  {
    user: 'root',
    pwd: 'secret',
    roles: [ { role: 'root', db: 'meal-maker-db' } ]
  }
);
```