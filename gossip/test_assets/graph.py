#!/bin/python3

import networkx as nx
import matplotlib.pyplot as plt
import json
import sys
import argparse
import os.path

seed = 20160  # seed random number generators for reproducibility

parser = argparse.ArgumentParser()
parser.add_argument("-n", "--nodes", type=int, help="The number of nodes", required=True)
parser.add_argument("-v", "--view", action="store_true", help="Set this to show the generated graph in the end.")
parser.add_argument("file", type=str, help="Path to which to write the json view of the graph. Pass - to write to stderr")
p_model = parser.add_subparsers(dest="model", title="model")

p_erdos = p_model.add_parser("erdos")
p_erdos.add_argument("-p", type=float, help="Probability for edge creation", required=True)

p_watts = p_model.add_parser("watts")
p_watts.add_argument("-p", type=float, help="The probability of rewiring each edge", required=True)
p_watts.add_argument("-k", type=float, help="Each node is joined with its k nearest neighbors in a ring topology", required=True)
p_watts.add_argument("-t", "--tries", type=int, help="how often try to generate a connected graph", required=True)

p_power = p_model.add_parser("powerlaw")
p_power.add_argument("-m", type=int, help="the number of random edges to add for each new node", required=True)
p_power.add_argument("-p", type=float, help="Probability of adding a triangle after adding a random edge", required=True)

p_barabasi = p_model.add_parser("barabasi")
p_barabasi.add_argument("-m", type=int, help="Number of edges to attach from a new node to existing nodes", required=True)

args = parser.parse_args()
assert args.file == "-" or (not os.path.exists(args.file+".json") and not os.path.exists(args.file+".dot"))

if args.model == "erdos":
    G = nx.erdos_renyi_graph(n=args.nodes, p=args.p, seed=seed)
elif args.model == "watts":
    G = nx.connected_watts_strogatz_graph(n=args.nodes, k=args.k, p=args.p, tries=args.tries, seed=seed)
elif args.model == "powerlaw":
    G = nx.powerlaw_cluster_graph(n=args.nodes, m=args.m, p=args.p, seed=seed)
elif args.model == "barabasi":
    G = nx.barabasi_albert_graph(n=args.nodes, m=args.m, seed=seed)
else:
    assert False

# some properties
print("node degree clustering")
for v in nx.nodes(G):
    print(f"{v} {nx.degree(G, v)} {nx.clustering(G, v)}", file=sys.stderr)

if args.file == "-":
    json.dump({'nodes': list(nx.nodes(G)), 'edges': list(nx.edges(G))}, sys.stderr)
    sys.stderr.write("\n")
else:
    with open(args.file+".json", "w") as f:
        json.dump({'nodes': list(nx.nodes(G)), 'edges': list(nx.edges(G))}, f)
    with open(args.file+".dot", "w") as f:
        print(r"strict graph{", file=f)
        for i,n in enumerate(nx.nodes(G)):
            print(f"{n}[class=\"_{i}\"];", file=f)
        for a,b in nx.edges(G):
            print(f"{a} -- {b};", file=f)
        print(r"}", file=f)

if args.view:
    pos = nx.spring_layout(G, seed=seed)  # Seed for reproducible layout
    nx.draw(G, pos=pos)
    plt.show()
