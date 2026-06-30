"""Export a trained LightGBM booster to a portable JSON the Go service scores.

LightGBM has no native Go runtime, so we serialise the decision-tree ensemble
into a flat, language-agnostic structure. Because all features are numeric, every
split is a ``feature <= threshold`` test, which the Go scorer evaluates by walking
each tree to a leaf and summing the leaf values. Applying the sigmoid to that sum
reproduces ``booster.predict`` for the binary objective.

A ``base_score`` term captures any constant initial prediction LightGBM folds in
(via ``boost_from_average``); the exporter computes it empirically and the parity
test asserts the Go scorer matches Python to within a tight tolerance.
"""

from __future__ import annotations

from typing import Any

import lightgbm as lgb
import numpy as np


def _flatten_tree(structure: dict[str, Any]) -> list[dict[str, Any]]:
    """Flatten one LightGBM tree (nested dict) into an indexed node array.

    Node 0 is the root. Internal nodes reference children by array index; leaves
    carry their output value.
    """
    nodes: list[dict[str, Any]] = []

    def visit(node: dict[str, Any]) -> int:
        idx = len(nodes)
        if "leaf_value" in node:
            nodes.append({"leaf": True, "value": float(node["leaf_value"])})
            return idx
        # Reserve this slot before recursing so children get later indices.
        nodes.append({})
        left = visit(node["left_child"])
        right = visit(node["right_child"])
        nodes[idx] = {
            "leaf": False,
            "feature": int(node["split_feature"]),
            "threshold": float(node["threshold"]),
            "default_left": bool(node["default_left"]),
            "left": left,
            "right": right,
        }
        return idx

    visit(structure)
    return nodes


def _leaf_sum(trees: list[list[dict[str, Any]]], x: np.ndarray) -> float:
    """Score a single feature vector by walking each flattened tree (NumPy-free path)."""
    total = 0.0
    for nodes in trees:
        i = 0
        while not nodes[i]["leaf"]:
            node = nodes[i]
            value = x[node["feature"]]
            if np.isnan(value):
                i = node["left"] if node["default_left"] else node["right"]
            elif value <= node["threshold"]:
                i = node["left"]
            else:
                i = node["right"]
        total += nodes[i]["value"]
    return total


def export_model(
    booster: lgb.Booster,
    feature_names: list[str],
    calibration_x: np.ndarray,
) -> dict[str, Any]:
    """Build the portable model dict.

    ``calibration_x`` is a sample of feature vectors used to derive ``base_score``
    by comparing LightGBM's raw prediction to the bare leaf-value sum.
    """
    dump = booster.dump_model()
    trees = [_flatten_tree(t["tree_structure"]) for t in dump["tree_info"]]

    raw_pred = booster.predict(calibration_x, raw_score=True)
    leaf_sums = np.array([_leaf_sum(trees, row) for row in calibration_x])
    deltas = raw_pred - leaf_sums
    base_score = float(np.mean(deltas))
    # The delta should be a constant init score; flag if it is not.
    if float(np.std(deltas)) > 1e-6:
        raise ValueError("non-constant base score; tree export does not reproduce raw prediction")

    return {
        "format_version": 1,
        "objective": "binary",  # Go applies a sigmoid to the raw score.
        "num_features": len(feature_names),
        "feature_names": list(feature_names),
        "base_score": base_score,
        "trees": [{"nodes": nodes} for nodes in trees],
    }
