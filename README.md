# devenv

Easily manage orion-ptt-system instances.

## Create a Stack

    [dbt] devenv create <name>

## Destroy a Stack

    [dbt] devenv destroy <name>

## Glass a Stack (Nuke and Pave)

    [dbt] devenv glass <name>

NB: Parameters in `CreateStack()` function must match current template in [https://orion-ptt-system.s3.amazonaws.com/orion-ptt-system.yaml](https://orion-ptt-system.s3.amazonaws.com/orion-ptt-system.yaml) else errors will be thrown.