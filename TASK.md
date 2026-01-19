# TASK MISSION

## Background
As a new employee you’ve just inherited an incomplete internal tool. This tool’s purpose is to help the SRE team with the goal of keeping the workloads on our Kubernetes cluster reliable and secure.

You are tasked with extending this tool according to the user stories below:
# SRE User Stories
> (Least 2)
- [X] As an SRE I want to know whether all the deployments in the k8s cluster have as many healthy pods as requested by the respective `Deployment` spec
- [ ] As an SRE I want to prevent two workloads defined by k8s namespace(s) and label selectors from being able to exchange any network activity on demand
- [X] As an SRE I want to always know whether this tool can successfully communicate with the configured k8s API server

# Application Developer Stories
> (Least 1)
- [X] As an application developer I want to build this application into a container image when I push a commit to the `main` branch of its repository
- [ ] As an application developer I want to be able to deploy this application into a Kubernetes cluster using Helm

---
The incomplete tool can be found here: https://github.com/TykTechnologies/tyk-sre-assignment
Clone the repository 
Choose your language of choice - available in Go and Python
- [X] Go
- [ ] Python

Complete at least 2 "As an SRE" and at least 1 "As an application developer" stories - extending the existing tool
Push your changes to your clone on Github and share with us before the day of your interview
Make sure all automated tests are passing before sharing the project.
