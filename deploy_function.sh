FUNCTION_NAME=$(cd terraform && terraform output queue-function-name)
(cd function && func azure functionapp publish "$FUNCTION_NAME")
