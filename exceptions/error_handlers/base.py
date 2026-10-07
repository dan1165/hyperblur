import inspect


def create_user_friendly_error_message(request, exception):
    # Taken from https://github.com/searxng/searxng/blob/f5eb56b63f250c7804e5e1cf4426e550bc933906/searx/metrics/error_recorder.py
    exception_class = exception.__class__
    exception_name = exception_class.__qualname__
    exception_module = exception_class.__module__

    if exception_module is None or exception_module == str.__class__.__module__:
        processed_exception_name = exception_name
    else:
        processed_exception_name = f"{exception_module}.{exception_name}"

    exception_message = str(exception) or None

    frame = inspect.trace()
    context = []

    for trace in reversed(frame):
        if trace.filename.startswith(request.app.ctx.OPENBLUR_PARENT_DIR_PATH):
            local_path = trace.filename[len(request.app.ctx.OPENBLUR_PARENT_DIR_PATH) + 1 :]
        else:
            local_path = trace.filename
        occurrence = f'File "{local_path}" line {trace.lineno}: in {trace.function}'

        # Add code context is applicable
        if trace.code_context:
            occurrence = f"{occurrence}\n  {trace.code_context[0].strip()}"

        context.append(occurrence)

    return processed_exception_name, exception_message, "\n".join(context)
