#!/usr/bin/env python3
"""Generate stateful link-aware API chains using Schemathesis."""

import argparse
import json
import numbers
import sys

import schemathesis
from schemathesis.specs.openapi.expressions import ExpressionContext
from schemathesis.specs.openapi.stateful import get_all_links, expand_status_code

RESPONSE_BODY_EXPR_PREFIX = "$response.body#"


def _parse_response_payload(response):
    try:
        return response.json()
    except Exception:
        try:
            text = response.text
        except Exception:
            return None
        if text is None:
            return None
        text = text.strip()
        return text if text else None


def _normalize_request_body(body):
    if body is None:
        return None
    if body.__class__.__name__ == "NotSet":
        return None
    return body


def _decode_json_pointer_token(token):
    return token.replace("~1", "/").replace("~0", "~")


def _parse_response_body_expression(expression):
    if not isinstance(expression, str):
        return None
    if not expression.startswith(RESPONSE_BODY_EXPR_PREFIX):
        return None
    pointer = expression[len(RESPONSE_BODY_EXPR_PREFIX) :]
    if pointer == "":
        return "/"
    if pointer.startswith("/"):
        return pointer
    return None


def _extract_json_pointer_from_expression(expression):
    return _parse_response_body_expression(expression)


def _resolve_response_expression(response_payload, expression):
    pointer = _parse_response_body_expression(expression)
    if pointer is None:
        return None
    if pointer == "/":
        return response_payload
    current = response_payload
    for raw_token in pointer.split("/")[1:]:
        token = _decode_json_pointer_token(raw_token)
        if isinstance(current, list):
            try:
                idx = int(token)
            except Exception:
                return None
            if idx < 0 or idx >= len(current):
                return None
            current = current[idx]
            continue
        if not isinstance(current, dict):
            return None
        if token not in current:
            return None
        current = current[token]
    return current


def _iter_expression_candidates(value):
    if isinstance(value, str):
        yield value
        return
    if isinstance(value, dict):
        for nested in value.values():
            yield from _iter_expression_candidates(nested)
        return
    if isinstance(value, (list, tuple)):
        for nested in value:
            yield from _iter_expression_candidates(nested)


def _extract_response_expression_pairs(link, response_payload):
    pairs = []
    seen = set()
    candidates = []
    parameters = getattr(link, "parameters", None)
    if parameters is not None:
        candidates.append(parameters)
    request_body = getattr(link, "request_body", None)
    if request_body is not None:
        candidates.append(request_body)
    body = getattr(link, "body", None)
    if body is not None:
        candidates.append(body)
    for source in candidates:
        for expression in _iter_expression_candidates(source):
            resolved = _resolve_response_expression(response_payload, expression)
            if resolved is None:
                continue
            pointer = _extract_json_pointer_from_expression(expression)
            if pointer is None:
                continue
            try:
                marker = json.dumps([pointer, resolved], sort_keys=True, default=str)
            except Exception:
                marker = f"{pointer}:{resolved}"
            if marker in seen:
                continue
            seen.add(marker)
            pairs.append((resolved, pointer))
    return pairs


def _is_numeric(value):
    return isinstance(value, numbers.Number) and not isinstance(value, bool)


def _rewrite_response_derived_values(payload, derived_pairs):
    if payload is None or not derived_pairs:
        return payload
    exact_lookup = {}
    numeric_lookup = {}

    for derived, pointer in derived_pairs:
        if _is_numeric(derived):
            numeric_lookup.setdefault(float(derived), pointer)
            continue
        if isinstance(derived, (str, bool, type(None))):
            exact_lookup.setdefault((type(derived), derived), pointer)

    def walk(node):
        if isinstance(node, dict):
            return {k: walk(v) for k, v in node.items()}
        if isinstance(node, list):
            return [walk(item) for item in node]
        if _is_numeric(node):
            pointer = numeric_lookup.get(float(node))
            if pointer is not None:
                return pointer
            return node
        if isinstance(node, (str, bool, type(None))):
            pointer = exact_lookup.get((type(node), node))
            if pointer is not None:
                return pointer
        return node

    return walk(payload)


def _log_debug(message, debug=False):
    if debug:
        print(message, file=sys.stderr)


def _log_call_failure(message, status=None, payload=None):
    details = message
    if status is not None:
        details += f" status={status}"
    if payload is not None:
        details += f" response={json.dumps(payload, default=str)}"
    print(details, file=sys.stderr)


class StepCallError(RuntimeError):
    def __init__(self, message, status=None, payload=None):
        super().__init__(message)
        self.status = status
        self.payload = payload


def _operation_key(method, path):
    return f"{method.upper()}:{path}"


def _parse_chain_nodes(chain_arg):
    if not isinstance(chain_arg, str) or not chain_arg.strip():
        raise RuntimeError("--chain must be a non-empty comma-separated list of operationIds")
    chain_nodes = [item.strip() for item in chain_arg.split(",") if item.strip()]
    if not chain_nodes:
        raise RuntimeError("--chain contains no valid operationIds")
    return chain_nodes


def _extract_operation_id(operation):
    raw = operation.definition.resolved
    if not isinstance(raw, dict):
        return None
    return raw.get("operationId")


def _build_operation_id_index(operations):
    operations_by_id = {}
    for operation in operations:
        operation_id = _extract_operation_id(operation)
        if not operation_id:
            continue
        if operation_id in operations_by_id:
            raise RuntimeError(f"duplicate OpenAPI operationId '{operation_id}' is not supported")
        operations_by_id[operation_id] = operation
    return operations_by_id

def _find_transition_link(previous_case, previous_response, next_operation_id):
    links = []
    for status_code, link in get_all_links(previous_case.operation):
        if previous_response.status_code in set(expand_status_code(status_code)):
            links.append(link)
    for link in links:
        target = link.get_target_operation()
        target_operation_id = _extract_operation_id(target)
        if target_operation_id != next_operation_id:
            continue
        return link, target
    return None, None


def _execute_case(case, base_url, node_name):
    try:
        response = case.call(base_url=base_url, timeout=10)
    except Exception as exc:
        raise StepCallError(
            f"call failed at chain step '{node_name}' method={case.method.upper()} path={case.formatted_path}: {exc}"
        ) from exc

    response_payload = _parse_response_payload(response)
    status = getattr(response, "status_code", None)
    if status is not None and (status < 200 or status >= 300):
        raise StepCallError(
            f"call failed at chain step '{node_name}' method={case.method.upper()} path={case.formatted_path}",
            status=status,
            payload=response_payload,
        )
    return response, response_payload, status


def _generate_case_once(operation, context_label, configure_case=None):
    try:
        case = operation.as_strategy(
            data_generation_method=schemathesis.DataGenerationMethod.positive
        ).example()
        if configure_case is not None:
            configure_case(case)
        return case
    except Exception as exc:
        raise RuntimeError(f"failed to generate case for {context_label}: {exc}") from exc


def _apply_linked_rewrite(value, derived_pairs, rewrite_linked_values):
    if not rewrite_linked_values:
        return value
    return _rewrite_response_derived_values(value, derived_pairs)


def _build_step_record(flow_id, case, status, response_payload, derived_pairs, rewrite_linked_values=True):
    op_raw = case.operation.definition.resolved
    operation_id = None
    if isinstance(op_raw, dict):
        operation_id = op_raw.get("operationId")
    request_body = _normalize_request_body(case.body)
    request_body = _apply_linked_rewrite(request_body, derived_pairs, rewrite_linked_values)
    path_parameters = _apply_linked_rewrite(case.path_parameters or {}, derived_pairs, rewrite_linked_values)
    query = _apply_linked_rewrite(case.query or {}, derived_pairs, rewrite_linked_values)
    headers = _apply_linked_rewrite(dict(case.headers or {}), derived_pairs, rewrite_linked_values)
    return {
        "iterationIndex": 0,
        "flowId": flow_id,
        "operationId": operation_id,
        "method": case.method.upper(),
        "pathTemplate": case.path,
        "resolvedPath": case.formatted_path,
        "pathParameters": path_parameters,
        "query": query,
        "headers": headers,
        "requestBody": request_body,
        "status": status,
        "responseBody": response_payload,
    }


def _format_request_debug_payload(case, derived_pairs, rewrite_linked_values=True):
    request_body = _normalize_request_body(case.body)
    request_body = _apply_linked_rewrite(request_body, derived_pairs, rewrite_linked_values)
    path_parameters = _apply_linked_rewrite(case.path_parameters or {}, derived_pairs, rewrite_linked_values)
    query = _apply_linked_rewrite(case.query or {}, derived_pairs, rewrite_linked_values)
    headers = _apply_linked_rewrite(dict(case.headers or {}), derived_pairs, rewrite_linked_values)
    return {
        "method": case.method.upper(),
        "resolvedPath": case.formatted_path,
        "pathParameters": path_parameters,
        "headers": headers,
        "query": query,
        "requestBody": request_body,
    }


def _run_stateful_chains(
    schema,
    base_url,
    chain_operation_ids,
    debug=False,
    max_tries=1,
    rewrite_linked_values=True,
):
    operations = []
    for result in schema.get_all_operations():
        operations.append(result.ok())
    operations_by_id = _build_operation_id_index(operations)
    operation_by_key = {}
    for op in operations:
        operation_by_key[_operation_key(op.method.upper(), op.path)] = op

    if not chain_operation_ids:
        raise RuntimeError("chain sequence is empty")
    for operation_id in chain_operation_ids:
        if operation_id not in operations_by_id:
            raise RuntimeError(f"operationId '{operation_id}' is not present in OpenAPI spec")

    steps = []
    previous_case = None
    previous_response = None
    previous_response_payload = None
    previous_operation_id = None

    for step_idx, operation_id in enumerate(chain_operation_ids):
        last_error = None
        last_status = None
        last_payload = None
        step_completed = False

        for attempt in range(1, max_tries + 1):
            _log_debug(
                f"[stateful-chain] step={step_idx} operationId={operation_id} attempt={attempt}/{max_tries}",
                debug=debug,
            )
            derived_pairs = []
            try:
                if step_idx == 0:
                    operation = operations_by_id[operation_id]
                    case = _generate_case_once(
                        operation,
                        context_label=f"first chain operationId '{operation_id}'",
                    )
                else:
                    link, target = _find_transition_link(previous_case, previous_response, operation_id)
                    if link is None or target is None:
                        raise RuntimeError(
                            f"cannot transition from '{previous_operation_id}' to '{operation_id}' using OpenAPI links"
                        )

                    def _configure_linked_case(
                        next_case, _link=link, _previous_response=previous_response, _previous_case=previous_case
                    ):
                        _link.set_data(
                            next_case,
                            elapsed=0.0,
                            context=ExpressionContext(response=_previous_response, case=_previous_case),
                        )

                    case = _generate_case_once(
                        target,
                        context_label=f"linked transition '{previous_operation_id}' -> '{operation_id}'",
                        configure_case=_configure_linked_case,
                    )
                    derived_pairs = _extract_response_expression_pairs(link, previous_response_payload)
                    _log_debug(
                        f"[stateful-chain] step={step_idx} transition={previous_operation_id}->{operation_id}",
                        debug=debug,
                    )

                response, response_payload, status = _execute_case(case, base_url, operation_id)
                steps.append(
                    _build_step_record(
                        operation_id,
                        case,
                        status,
                        response_payload,
                        derived_pairs,
                        rewrite_linked_values=rewrite_linked_values,
                    )
                )
                previous_case = case
                previous_response = response
                previous_response_payload = response_payload
                previous_operation_id = operation_id
                step_completed = True
                break
            except StepCallError as exc:
                last_error = exc
                last_status = exc.status
                last_payload = exc.payload
                if exc.status is not None:
                    request_payload = _format_request_debug_payload(
                        case,
                        derived_pairs,
                        rewrite_linked_values=rewrite_linked_values,
                    )
                    _log_debug(
                        f"[stateful] non-2xx chainStep={operation_id} attempt={attempt}/{max_tries} "
                        f"request={json.dumps(request_payload, default=str)} "
                        f"status={exc.status} response={json.dumps(exc.payload, default=str)}",
                        debug=debug,
                    )
                else:
                    _log_debug(
                        f"[stateful] call exception chainStep={operation_id} attempt={attempt}/{max_tries} error={exc}",
                        debug=debug,
                    )
            except Exception as exc:
                last_error = exc
                _log_debug(
                    f"[stateful-chain] step={step_idx} operationId={operation_id} failed attempt={attempt}/{max_tries} error={exc}",
                    debug=debug,
                )

        if not step_completed:
            if last_status is not None:
                raise RuntimeError(
                    f"call failed at chain step '{operation_id}' after {max_tries} attempt(s) "
                    f"status={last_status} response={json.dumps(last_payload, default=str)}"
                ) from last_error
            raise RuntimeError(
                f"call failed at chain step '{operation_id}' after {max_tries} attempt(s): {last_error}"
            ) from last_error

    return [{"iterationIndex": 0, "chainIndex": 0, "steps": steps}]


def main():
    parser = argparse.ArgumentParser(
        description="Generate stateful link-aware chains using Schemathesis"
    )
    parser.add_argument(
        "--openapi-link",
        required=False,
        help="OpenAPI path or URL",
    )
    parser.add_argument(
        "--output", required=True, help="Path to write the output JSON file"
    )
    parser.add_argument(
        "--base-url",
        default="http://localhost:9966/petclinic/api",
        help="Base URL for stateful API calls",
    )
    parser.add_argument(
        "--chain",
        required=True,
        help="Ordered comma-separated operationIds (single explicit chain)",
    )
    parser.add_argument(
        "--debug",
        action="store_true",
        help="Enable debug stderr logs (including WRR choices)",
    )
    parser.add_argument(
        "--max-tries",
        type=int,
        default=1,
        help="Maximum full-step attempts per chain step (generation/linking/call) until success",
    )
    parser.add_argument(
        "--no-rewrite-linked-values",
        action="store_true",
        help="Disable replacing linked values with JSON pointers in emitted output/debug request payloads",
    )

    args = parser.parse_args()
    openapi_link = args.openapi_link
    if not openapi_link:
        print("Error: --openapi-link is required", file=sys.stderr)
        sys.exit(1)
    if args.max_tries < 1:
        print("Error: --max-tries must be >= 1", file=sys.stderr)
        sys.exit(1)

    # Load the OpenAPI spec
    try:
        if openapi_link.startswith(("http://", "https://")):
            schema = schemathesis.from_uri(openapi_link)
        else:
            schema = schemathesis.from_path(openapi_link)
    except Exception as e:
        print(f"Error loading OpenAPI spec: {e}", file=sys.stderr)
        sys.exit(1)

    try:
        chain_operation_ids = _parse_chain_nodes(args.chain)
        chains = _run_stateful_chains(
            schema,
            args.base_url,
            chain_operation_ids,
            debug=args.debug,
            max_tries=args.max_tries,
            rewrite_linked_values=not args.no_rewrite_linked_values,
        )
    except Exception as e:
        print(f"Error generating stateful chains: {e}", file=sys.stderr)
        sys.exit(1)

    # Write output
    try:
        with open(args.output, "w") as f:
            json.dump(chains, f, indent=2, default=str)
    except Exception as e:
        print(f"Error writing output file: {e}", file=sys.stderr)
        sys.exit(1)

    print(f"Generated {len(chains)} stateful chain(s) to {args.output}")

if __name__ == "__main__":
    main()

