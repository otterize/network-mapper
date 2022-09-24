# The only purpose this Dockerfile serves is to be able to run buildx to push a multi-platform image without rebuilding.

ARG SOURCE_IMAGE
FROM $SOURCE_IMAGE