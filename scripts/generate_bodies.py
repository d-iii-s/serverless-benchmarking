#!/usr/bin/env python3
"""Generate random request bodies for a specific OpenAPI endpoint using Schemathesis.

Usage:
    python3 generate_bodies.py \
        --spec-path ./openapi.yaml \
        --endpoint /pets \
        --method POST \
        --count 5 \
        --output ./bodies.json
"""

import argparse
import json
import sys

import schemathesis
from hypothesis import given, settings, HealthCheck


def main():
    parser = argparse.ArgumentParser(
        description="Generate request bodies from an OpenAPI specification using Schemathesis"
    )
    parser.add_argument(
        "--spec-path", required=True, help="Path to the OpenAPI specification file"
    )
    parser.add_argument(
        "--endpoint", required=True, help="API endpoint path (e.g. /pets)"
    )
    parser.add_argument(
        "--method",
        required=True,
        help="HTTP method (e.g. POST)",
    )
    parser.add_argument(
        "--count",
        type=int,
        required=True,
        help="Number of request bodies to generate",
    )
    parser.add_argument(
        "--output", required=True, help="Path to write the output JSON file"
    )

    args = parser.parse_args()

    method = args.method.upper()

    # Load the OpenAPI spec
    try:
        schema = schemathesis.from_path(args.spec_path)
    except Exception as e:
        print(f"Error loading OpenAPI spec: {e}", file=sys.stderr)
        sys.exit(1)

    # Find the matching operation.
    # get_all_operations() yields Result-wrapped objects; unwrap via .ok().
    operation = None
    for result in schema.get_all_operations():
        op = result.ok()
        if op.path == args.endpoint and op.method.upper() == method:
            operation = op
            break

    if operation is None:
        print(
            f"Error: no operation found for {method} {args.endpoint}",
            file=sys.stderr,
        )
        sys.exit(1)

    # Guard: only accept application/json request bodies.
    content_types = operation.get_request_payload_content_types()
    if content_types and "application/json" not in content_types:
        print(
            f"Error: operation {method} {args.endpoint} does not support "
            f"application/json request body (available: {content_types})",
            file=sys.stderr,
        )
        sys.exit(1)

    strategy = operation.as_strategy()

    # Collect generated bodies
    bodies = []
    target_count = args.count

    @given(case=strategy)
    @settings(
        max_examples=target_count,
        suppress_health_check=[HealthCheck.too_slow],
        deadline=None,
    )
    def collect(case):
        body = case.body
        if body is not None:
            bodies.append(body)

    try:
        collect()
    except Exception as e:
        print(f"Error generating test cases: {e}", file=sys.stderr)
        sys.exit(1)

    if not bodies:
        print(
            f"Warning: no request bodies generated for {method} {args.endpoint}. "
            "The endpoint may not have a request body schema.",
            file=sys.stderr,
        )

    # Write output
    try:
        with open(args.output, "w") as f:
            json.dump(bodies, f, indent=2, default=str)
    except Exception as e:
        print(f"Error writing output file: {e}", file=sys.stderr)
        sys.exit(1)

    print(f"Generated {len(bodies)} request body/bodies to {args.output}")


if __name__ == "__main__":
    main()

