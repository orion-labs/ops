# devenv

Easily manage orion-ptt-system instances.

## Create a Dev Environment

    [dbt] devenv create <name>

## Destroy a Dev Environment

    [dbt] devenv destroy <name>

## Display Status of a Dev Environment

    [dbt] devenv status <name>

## Glass a Dev Environment (Nuke and Pave)

    [dbt] devenv glass <name>

NB: Parameters in `CreateStack()` function must match current template in [https://orion-ptt-system.s3.amazonaws.com/orion-ptt-system.yaml](https://orion-ptt-system.s3.amazonaws.com/orion-ptt-system.yaml) else errors will be thrown.