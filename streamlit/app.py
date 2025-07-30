import streamlit as st
import requests
import pandas as pd
import plotly.graph_objects as go
import plotly.express as px
from datetime import datetime, timedelta
import time
import json

# Configuration
API_BASE_URL = "http://localhost:8080/api"

# Page configuration
st.set_page_config(
    page_title="Basis Trading Monitor",
    page_icon="📊",
    layout="wide",
    initial_sidebar_state="expanded"
)

# Custom CSS
st.markdown("""
<style>
    .stMetric {
        background-color: #f0f2f6;
        padding: 10px;
        border-radius: 5px;
        border: 1px solid #ddd;
    }
    .metric-container {
        display: flex;
        justify-content: space-between;
        margin-bottom: 20px;
    }
</style>
""", unsafe_allow_html=True)

# Title and description
st.title("🚀 Basis Trading Monitor")
st.markdown("Real-time monitoring dashboard for cryptocurrency basis trading strategies")

# Sidebar
with st.sidebar:
    st.header("Control Panel")
    
    # Auto-refresh toggle
    auto_refresh = st.checkbox("Auto Refresh", value=True)
    refresh_interval = st.slider("Refresh Interval (seconds)", 1, 60, 5)
    
    # Strategy Management
    st.subheader("Strategy Management")
    
    with st.form("new_strategy"):
        st.write("Add New Strategy")
        spot_symbol = st.text_input("Spot Symbol", "BTC-USD")
        future_symbol = st.text_input("Future Symbol", "BTC-PERP")
        target_basis = st.number_input("Target Basis (%)", min_value=0.0, value=5.0, step=0.1)
        max_position = st.number_input("Max Position Size", min_value=0.0, value=1.0, step=0.1)
        min_trade_size = st.number_input("Min Trade Size", min_value=0.0, value=0.01, step=0.001)
        
        if st.form_submit_button("Add Strategy"):
            strategy_data = {
                "spot_symbol": spot_symbol,
                "future_symbol": future_symbol,
                "target_basis": target_basis,
                "max_position": max_position,
                "min_trade_size": min_trade_size,
                "is_active": True
            }
            try:
                response = requests.post(f"{API_BASE_URL}/strategies", json=strategy_data)
                if response.status_code == 201:
                    st.success("Strategy added successfully!")
                else:
                    st.error(f"Failed to add strategy: {response.text}")
            except Exception as e:
                st.error(f"Error: {str(e)}")

# Helper functions
def fetch_data(endpoint):
    """Fetch data from API endpoint"""
    try:
        response = requests.get(f"{API_BASE_URL}/{endpoint}")
        if response.status_code == 200:
            return response.json()
        else:
            st.error(f"Failed to fetch {endpoint}: {response.status_code}")
            return []
    except Exception as e:
        st.error(f"Error fetching {endpoint}: {str(e)}")
        return []

def create_basis_chart(snapshots_df):
    """Create basis percentage chart"""
    fig = go.Figure()
    
    for symbol_pair in snapshots_df['pair'].unique():
        pair_data = snapshots_df[snapshots_df['pair'] == symbol_pair]
        fig.add_trace(go.Scatter(
            x=pair_data['timestamp'],
            y=pair_data['basis_percent'],
            mode='lines+markers',
            name=symbol_pair,
            line=dict(width=2),
            marker=dict(size=6)
        ))
    
    fig.update_layout(
        title="Basis Percentage Over Time",
        xaxis_title="Time",
        yaxis_title="Basis %",
        height=400,
        hovermode='x unified'
    )
    
    return fig

def create_position_chart(positions_df):
    """Create position size chart"""
    fig = px.bar(
        positions_df,
        x='symbol',
        y='size',
        color='side',
        title="Current Positions",
        height=300,
        color_discrete_map={'long': '#00CC88', 'short': '#FF4444'}
    )
    
    return fig

# Main content area
col1, col2, col3, col4 = st.columns(4)

# Fetch current data
if 'last_update' not in st.session_state:
    st.session_state.last_update = time.time()

if auto_refresh and (time.time() - st.session_state.last_update) > refresh_interval:
    st.session_state.last_update = time.time()
    st.rerun()

# Fetch data
health = fetch_data("health")
snapshots = fetch_data("basis/snapshots")
strategies = fetch_data("strategies")
positions = fetch_data("positions")
trades = fetch_data("trades")

# Display health status
if health:
    with col1:
        st.metric("System Status", health.get('status', 'Unknown').upper(), 
                 delta="Online" if health.get('status') == 'healthy' else "Offline")

# Display key metrics
if snapshots:
    latest_snapshot = snapshots[0] if snapshots else {}
    with col2:
        st.metric("Latest Basis", f"{latest_snapshot.get('basis_percent', 0):.2f}%")
    with col3:
        st.metric("Spot Price", f"${latest_snapshot.get('spot_price', 0):,.2f}")
    with col4:
        st.metric("Future Price", f"${latest_snapshot.get('future_price', 0):,.2f}")

# Basis Chart
st.subheader("📈 Basis Analysis")
if snapshots:
    # Convert to DataFrame for easier manipulation
    snapshots_df = pd.DataFrame(snapshots)
    if not snapshots_df.empty:
        snapshots_df['timestamp'] = pd.to_datetime(snapshots_df['timestamp'])
        snapshots_df['pair'] = snapshots_df['spot_symbol'] + '/' + snapshots_df['future_symbol']
        
        basis_chart = create_basis_chart(snapshots_df)
        st.plotly_chart(basis_chart, use_container_width=True)
else:
    st.info("No basis data available yet")

# Two column layout for positions and strategies
col_left, col_right = st.columns(2)

with col_left:
    st.subheader("📊 Active Positions")
    if positions:
        positions_df = pd.DataFrame(positions)
        position_chart = create_position_chart(positions_df)
        st.plotly_chart(position_chart, use_container_width=True)
        
        # Position details table
        st.dataframe(
            positions_df[['symbol', 'side', 'size', 'entry_price', 'mark_price', 'unrealized_pl']],
            use_container_width=True
        )
    else:
        st.info("No active positions")

with col_right:
    st.subheader("🎯 Active Strategies")
    if strategies:
        strategies_df = pd.DataFrame(strategies)
        st.dataframe(
            strategies_df[['id', 'spot_symbol', 'future_symbol', 'target_basis', 'max_position', 'is_active']],
            use_container_width=True
        )
    else:
        st.info("No active strategies")

# Recent trades section
st.subheader("💹 Recent Trades")
if trades:
    trades_df = pd.DataFrame(trades)
    trades_df['created_at'] = pd.to_datetime(trades_df['created_at'])
    
    # Display recent trades
    st.dataframe(
        trades_df[['id', 'strategy_id', 'side', 'size', 'spot_price', 'future_price', 'basis', 'status', 'created_at']].tail(10),
        use_container_width=True
    )
else:
    st.info("No trades executed yet")

# Footer with last update time
st.markdown("---")
st.caption(f"Last updated: {datetime.now().strftime('%Y-%m-%d %H:%M:%S')}")